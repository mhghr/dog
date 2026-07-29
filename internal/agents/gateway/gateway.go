package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "monitoring-platform/internal/agent_gateway/pb"
	"monitoring-platform/internal/domain"
)

type GatewayConfig struct {
	ListenAddress string
	HealthAddress string
	TLSCertFile   string
	TLSKeyFile    string
	CACertFile    string
	DatabaseURL   string
}

type Gateway struct {
	cfg    GatewayConfig
	logger *slog.Logger
	rdb    *redis.Client

	mu      sync.Mutex
	agents  map[string]*agentConn
}

type agentConn struct {
	stream   pb.AgentGateway_AgentStreamServer
	agentID  string
	location string
	cancel   context.CancelFunc
}

func New(cfg GatewayConfig, logger *slog.Logger) (*Gateway, error) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	return &Gateway{
		cfg:    cfg,
		logger: logger,
		rdb:    rdb,
		agents: make(map[string]*agentConn),
	}, nil
}

func (g *Gateway) Close() {
	g.rdb.Close()
}

func (g *Gateway) ListenAndServe(ctx context.Context) error {
	lis, err := net.Listen("tcp", g.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer()
	pb.RegisterAgentGatewayServer(srv, g)

	g.logger.Info("gateway listening", "address", g.cfg.ListenAddress)

	go g.consumeJobs(ctx)

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}

func (g *Gateway) AgentStream(stream pb.AgentGateway_AgentStreamServer) error {
	// First message must contain the agent_id
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("receive registration: %w", err)
	}
	if msg.AgentId == "" {
		return fmt.Errorf("registration missing agent_id")
	}

	conn := &agentConn{
		stream:  stream,
		agentID: msg.AgentId,
	}

	ctx, cancel := context.WithCancel(stream.Context())
	conn.cancel = cancel

	g.mu.Lock()
	g.agents[msg.AgentId] = conn
	g.mu.Unlock()

	g.logger.Info("agent connected", "agent_id", msg.AgentId)

	defer func() {
		cancel()
		g.mu.Lock()
		delete(g.agents, msg.AgentId)
		g.mu.Unlock()
		g.logger.Info("agent disconnected", "agent_id", msg.AgentId)
	}()

	// Process incoming messages from agent
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			g.handleAgentMessage(ctx, msg)
		}
	}()

	<-ctx.Done()
	return nil
}

func (g *Gateway) handleAgentMessage(ctx context.Context, msg *pb.AgentMessage) {
	switch {
	case msg.Result != nil:
		g.handleResult(ctx, msg.AgentId, msg.Result)
	case msg.Heartbeat != nil:
		g.handleHeartbeat(ctx, msg.AgentId, msg.Heartbeat)
	}
}

func (g *Gateway) handleResult(ctx context.Context, agentID string, r *pb.ProbeResult) {
	metrics := make(map[string]any)
	attrs := make(map[string]any)
	if len(r.Metrics) > 0 {
		json.Unmarshal(r.Metrics, &metrics)
	}
	if len(r.Attributes) > 0 {
		json.Unmarshal(r.Attributes, &attrs)
	}

	result := domain.ProbeResult{
		ID:              r.Id,
		JobID:           r.JobId,
		MonitorID:       r.MonitorId,
		ProbeLocationID: r.ProbeLocationId,
		Success:         r.Success,
		ErrorCode:       r.ErrorCode,
		ErrorMessage:    r.ErrorMessage,
		DurationMillis:  r.DurationMillis,
		Metrics:         metrics,
		Attributes:      attrs,
	}

	// Send result to API
	g.sendToAPI(ctx, result)
}

func (g *Gateway) handleHeartbeat(ctx context.Context, agentID string, hb *pb.Heartbeat) {
	g.logger.Debug("heartbeat", "agent_id", agentID, "cpu", hb.CpuUsage, "running_jobs", hb.RunningJobs)
	// Update agent status in DB
}

func (g *Gateway) sendToAPI(ctx context.Context, result domain.ProbeResult) {
	// Forward to API ingestion endpoint
	payload, _ := json.Marshal(result)
	g.rdb.Publish(ctx, "probe_results", payload)
	g.logger.Info("result forwarded", "monitor_id", result.MonitorID, "success", result.Success)
}

func (g *Gateway) consumeJobs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := g.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    "gateway_workers",
			Consumer: "gateway-01",
			Streams:  []string{"probe_jobs", ">"},
			Count:    10,
			Block:    0,
		}).Result()
		if err != nil {
			g.logger.Error("read jobs", "error", err)
			continue
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				var raw string
				if v, ok := msg.Values["payload"]; ok {
					raw, _ = v.(string)
				}
				if raw == "" {
					continue
				}

				var job domain.ProbeJob
				if err := json.Unmarshal([]byte(raw), &job); err != nil {
					g.logger.Error("unmarshal job", "error", err)
					continue
				}

				g.dispatchJob(ctx, job)
				g.rdb.XAck(ctx, "probe_jobs", "gateway_workers", msg.ID)
			}
		}
	}
}

func (g *Gateway) dispatchJob(ctx context.Context, job domain.ProbeJob) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Find an agent in the right location
	for _, conn := range g.agents {
		// Send job to first available agent (location matching TODO)
		pbJob := &pb.ProbeJob{
			Id:              job.ID,
			MonitorId:       job.MonitorID,
			Type:            string(job.Type),
			Target:          job.Target,
			TimeoutMillis:   int32(job.TimeoutMillis),
			Retries:         int32(job.Retries),
			ProbeLocationId: job.ProbeLocationID,
		}
		if cfg, err := json.Marshal(job.Config); err == nil {
			pbJob.Config = cfg
		}

		if err := conn.stream.Send(&pb.GatewayMessage{Job: pbJob}); err != nil {
			g.logger.Error("send job to agent", "agent_id", conn.agentID, "error", err)
			continue
		}
		g.logger.Info("job dispatched", "job_id", job.ID, "agent_id", conn.agentID)
		break
	}
}


