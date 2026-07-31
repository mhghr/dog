package signer

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"go.opentelemetry.io/proto/otlp/collector/metrics/v1"

	"monitoring-platform/internal/monitoring/credential"
)

type fakeMetricsService struct {
	v1.UnimplementedMetricsServiceServer
	mu       sync.Mutex
	captured metadata.MD
}

func (f *fakeMetricsService) Export(ctx context.Context, _ *v1.ExportMetricsServiceRequest) (*v1.ExportMetricsServiceResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	f.mu.Lock()
	f.captured = md
	f.mu.Unlock()
	return &v1.ExportMetricsServiceResponse{}, nil
}

func TestProxyExportSignsRequests(t *testing.T) {
	credMgr, err := credential.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := credMgr.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}
	if _, err := credMgr.LoadCredentials(); err != nil {
		t.Fatal(err)
	}

	fake := &fakeMetricsService{}
	server := grpc.NewServer()
	v1.RegisterMetricsServiceServer(server, fake)

	lis := bufconn.Listen(1024 * 1024)
	defer lis.Close()
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("fake server serve error: %v", err)
		}
	}()
	defer server.Stop()

	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough:///fake-ingest",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	proxy, err := New(conn, credMgr, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()

	resp, err := proxy.Export(ctx, &v1.ExportMetricsServiceRequest{})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if resp == nil {
		t.Fatal("Export returned nil response")
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if got := fake.captured.Get("x-agent-id"); len(got) != 1 || got[0] != "ag_1" {
		t.Errorf("x-agent-id = %v, want [ag_1]", got)
	}
	if got := fake.captured.Get("x-signature"); len(got) != 1 || got[0] == "" {
		t.Errorf("x-signature = %v, want one non-empty value", got)
	}
	if got := fake.captured.Get("x-timestamp"); len(got) != 1 || got[0] == "" {
		t.Errorf("x-timestamp = %v, want one non-empty value", got)
	}
}
