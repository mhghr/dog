package metrics

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"log/slog"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/health"
	"monitoring-platform/packages/shared/probe"
	"monitoring-platform/packages/shared/security"
)

// recordingHealthRepo is an in-memory health.Repository that records every
// upserted parameter health state so the test can assert the engine outcome.
type recordingHealthRepo struct {
	states map[string]health.HealthState
}

func newRecordingHealthRepo() *recordingHealthRepo {
	return &recordingHealthRepo{states: map[string]health.HealthState{}}
}

func (r *recordingHealthRepo) ListParameterCatalog(context.Context, string) ([]health.ParameterDefinition, error) {
	return nil, nil
}
func (r *recordingHealthRepo) GetParameterDefinition(context.Context, string, string) (health.ParameterDefinition, error) {
	return health.ParameterDefinition{}, domain.ErrNotFound
}
func (r *recordingHealthRepo) ListParameterRules(context.Context, string) ([]health.ParameterRule, error) { return nil, nil }
func (r *recordingHealthRepo) GetParameterRule(context.Context, string, string) (health.ParameterRule, error) {
	return health.ParameterRule{}, domain.ErrNotFound
}
func (r *recordingHealthRepo) UpsertParameterRule(context.Context, *health.ParameterRule) error { return nil }
func (r *recordingHealthRepo) DeleteParameterRule(context.Context, string, string) error        { return nil }
func (r *recordingHealthRepo) UpsertHealthState(_ context.Context, state *health.ParameterHealthState) error {
	r.states[state.ParameterKey] = state.CurrentState
	return nil
}
func (r *recordingHealthRepo) GetHealthState(context.Context, string, string) (health.ParameterHealthState, error) {
	return health.ParameterHealthState{}, domain.ErrNotFound
}
func (r *recordingHealthRepo) ListHealthStates(context.Context, string) ([]health.ParameterHealthState, error) {
	return nil, nil
}
func (r *recordingHealthRepo) ListNotificationChannels(context.Context) ([]health.HealthNotificationChannel, error) {
	return nil, nil
}
func (r *recordingHealthRepo) GetNotificationChannel(context.Context, string) (health.HealthNotificationChannel, error) {
	return health.HealthNotificationChannel{}, domain.ErrNotFound
}
func (r *recordingHealthRepo) CreateNotificationChannel(context.Context, *health.HealthNotificationChannel) error {
	return nil
}
func (r *recordingHealthRepo) UpdateNotificationChannel(context.Context, *health.HealthNotificationChannel) error {
	return nil
}
func (r *recordingHealthRepo) DeleteNotificationChannel(context.Context, string) error { return nil }
func (r *recordingHealthRepo) ListNotificationPolicies(context.Context, string) ([]health.NotificationPolicy, error) {
	return nil, nil
}
func (r *recordingHealthRepo) GetNotificationPolicy(context.Context, string) (health.NotificationPolicy, error) {
	return health.NotificationPolicy{}, domain.ErrNotFound
}
func (r *recordingHealthRepo) CreateNotificationPolicy(context.Context, *health.NotificationPolicy) error {
	return nil
}
func (r *recordingHealthRepo) UpdateNotificationPolicy(context.Context, *health.NotificationPolicy) error {
	return nil
}
func (r *recordingHealthRepo) DeleteNotificationPolicy(context.Context, string) error { return nil }

// integrationJob builds the ProbeJob exactly the way the scheduler does for a
// resource-bound monitor: target = resource target, IDs for the resource,
// workspace and probe location.
func integrationJob(monitorType domain.MonitorType, target string, config map[string]any) domain.ProbeJob {
	if config == nil {
		config = map[string]any{}
	}
	return domain.ProbeJob{
		ID:              "job-1",
		MonitorID:       "monitor-1",
		ResourceID:      "resource-1",
		WorkspaceID:     "workspace-1",
		Type:            monitorType,
		Target:          target,
		TimeoutMillis:   5000,
		Retries:         1,
		Config:          config,
		ProbeLocationID: "loc-amsterdam",
		ScheduledAt:     time.Now().UTC(),
	}
}

// integrationDeps wires a Guard that allows loopback so the pipeline test can
// exercise the happy/failure paths against local test servers. The SSRF
// policy itself is asserted by the dedicated executor security tests.
func integrationDeps() probe.Deps {
	return probe.Deps{
		Guard:  security.NewGuard(true),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// runPipeline executes a job through the scheduler-shaped job → registry
// executor → retry layer, evaluates the result in the health engine, and
// serializes it for VictoriaMetrics.
func runPipeline(t *testing.T, job domain.ProbeJob) (domain.ProbeResult, *recordingHealthRepo, *health.Engine, []string) {
	t.Helper()

	registry := probe.DefaultRegistry(integrationDeps())
	executor, ok := registry.Get(job.Type)
	if !ok {
		t.Fatalf("no executor registered for %s", job.Type)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := probe.ExecuteWithRetry(ctx, executor, job)

	repo := newRecordingHealthRepo()
	engine := health.NewEngine(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := engine.EvaluateResult(ctx, &result)
	if err != nil {
		t.Fatalf("health evaluation failed: %v", err)
	}

	lines := buildLines(&result, string(job.Type), "amsterdam")
	return result, repo, engine, lines
}

// reachabilityState evaluates a BOOLEAN_FAILURE parameter through the health
// engine exactly as ingestion would, so the test observes the reachability
// outcome (boolean-failure states are computed, not persisted). The SSL
// catalog lives under the "tls" probe type while its parameter keys use the
// ssl.* prefix.
func reachabilityState(engine *health.Engine, key string, value float64) health.HealthState {
	monitorType := strings.SplitN(key, ".", 2)[0]
	if monitorType == "ssl" {
		monitorType = "tls"
	}
	var def health.ParameterDefinition
	for _, candidate := range health.AllParameters[monitorType] {
		if candidate.Key == key {
			def = candidate
			break
		}
	}
	if def.Key == "" {
		panic("catalog definition not found: " + key)
	}
	rule := health.ParameterRule{
		MonitorID:         "monitor-1",
		ParameterKey:      key,
		Mode:              health.ModeInheritDefault,
		Aggregation:       "avg",
		MissingDataPolicy: "IGNORE",
		Enabled:           true,
	}
	state, _ := engine.EvaluateParameter(context.Background(), "monitor-1", key, []float64{value}, &rule, def)
	return state
}

func startTCPListenerForTest(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func TestPipelineTCPResourceMonitor(t *testing.T) {
	listener := startTCPListenerForTest(t)

	// Resource: api.example.com with a TCP 443 monitoring type.
	job := integrationJob(domain.MonitorTCP, listener.Addr().String(), map[string]any{
		"port":       0, // target carries the port
		"ip_version": "ipv4",
	})

	result, repo, engine, lines := runPipeline(t, job)
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}

	// Health Engine: reachability up → OK (BOOLEAN_FAILURE), connect time
	// evaluated and persisted (HIGHER_IS_WORSE).
	if reachabilityState(engine, "tcp.reachability", 1) != health.HealthOK {
		t.Fatalf("expected tcp.reachability OK")
	}
	if repo.states["tcp.connect_time_ms"] != health.HealthOK {
		t.Fatalf("expected tcp.connect_time_ms OK, got %v", repo.states["tcp.connect_time_ms"])
	}

	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		`monitor_tcp_reachability{monitor_id="monitor-1",monitor_type="tcp",probe_location="amsterdam",resource_id="resource-1",workspace_id="workspace-1"} 1`,
		`monitor_tcp_connect_time_ms{`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected line %q in:\n%s", want, joined)
		}
	}
}

func TestPipelineTCPFailureFiresHealth(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	address := listener.Addr().String()
	listener.Close()

	job := integrationJob(domain.MonitorTCP, address, nil)

	result, _, engine, lines := runPipeline(t, job)
	if result.Success {
		t.Fatal("expected failure for closed port")
	}
	if result.ErrorCode != "connection_refused" && result.ErrorCode != "connection_failed" {
		t.Fatalf("unexpected error code %s", result.ErrorCode)
	}

	// Health Engine: reachability down → HealthError (BOOLEAN_FAILURE).
	if reachabilityState(engine, "tcp.reachability", 0) != health.HealthError {
		t.Fatalf("expected tcp.reachability HealthError")
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, `monitor_tcp_reachability{monitor_id="monitor-1",monitor_type="tcp",probe_location="amsterdam",resource_id="resource-1",workspace_id="workspace-1"} 0`) {
		t.Fatalf("expected down reachability line in:\n%s", joined)
	}
	if strings.Contains(joined, "monitor_tcp_connect_time_ms") {
		t.Fatalf("failed connection must not emit connect_time_ms, got:\n%s", joined)
	}
}

func startDNSServerForTest(t *testing.T, handler dns.HandlerFunc) string {
	t.Helper()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	server := &dns.Server{PacketConn: packetConn, Handler: handler}
	go func() { _ = server.ActivateAndServe() }()
	t.Cleanup(func() { _ = server.Shutdown() })
	time.Sleep(20 * time.Millisecond)
	return packetConn.LocalAddr().String()
}

func TestPipelineDNSResourceMonitor(t *testing.T) {
	server := startDNSServerForTest(t, func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)
		if record, err := dns.NewRR("example.org. 300 IN A 203.0.113.10"); err == nil {
			response.Answer = append(response.Answer, record)
		}
		_ = w.WriteMsg(response)
	})

	job := integrationJob(domain.MonitorDNS, "example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
		"expected_values": []any{"203.0.113.10"},
	})

	result, repo, engine, lines := runPipeline(t, job)
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}

	if reachabilityState(engine, "dns.reachability", 1) != health.HealthOK {
		t.Fatalf("expected dns.reachability OK")
	}
	if reachabilityState(engine, "dns.expected_record_match", 1) != health.HealthOK {
		t.Fatalf("expected dns.expected_record_match OK")
	}
	if repo.states["dns.response_time_ms"] != health.HealthOK {
		t.Fatalf("expected dns.response_time_ms OK, got %v", repo.states["dns.response_time_ms"])
	}

	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		`monitor_dns_reachability{monitor_id="monitor-1",monitor_type="dns",probe_location="amsterdam",resource_id="resource-1",workspace_id="workspace-1"} 1`,
		"monitor_dns_response_time_ms{",
		"monitor_dns_answer_count{monitor_id=",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected line %q in:\n%s", want, joined)
		}
	}
}

func TestPipelineDNSNXDOMAINFiresHealth(t *testing.T) {
	server := startDNSServerForTest(t, func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetRcode(r, dns.RcodeNameError)
		_ = w.WriteMsg(response)
	})

	job := integrationJob(domain.MonitorDNS, "missing.example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
	})

	result, _, engine, lines := runPipeline(t, job)
	if result.Success || result.ErrorCode != "nxdomain" {
		t.Fatalf("expected nxdomain failure, got %+v", result)
	}
	if reachabilityState(engine, "dns.reachability", 0) != health.HealthError {
		t.Fatalf("expected dns.reachability HealthError")
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, `monitor_dns_reachability{monitor_id="monitor-1",monitor_type="dns",probe_location="amsterdam",resource_id="resource-1",workspace_id="workspace-1"} 0`) {
		t.Fatalf("expected down reachability line in:\n%s", joined)
	}
}

func startTLSServerForTest(t *testing.T, cert tls.Certificate) (string, *x509.Certificate) {
	t.Helper()
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				buffer := make([]byte, 1)
				_, _ = conn.Read(buffer)
				conn.Close()
			}(conn)
		}
	}()
	t.Cleanup(func() { _ = listener.Close() })

	host, port, _ := net.SplitHostPort(listener.Addr().String())
	return host + ":" + port, leaf
}

func makeCertForTest(t *testing.T, notBefore, notAfter time.Time) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		DNSNames:     []string{"localhost"},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func tlsJobForIntegration(hostport string, config map[string]any) domain.ProbeJob {
	host, port, _ := net.SplitHostPort(hostport)
	if config == nil {
		config = map[string]any{}
	}
	config["port"] = port
	return integrationJob(domain.MonitorTLS, host, config)
}

func TestPipelineTLSResourceMonitor(t *testing.T) {
	now := time.Now()
	cert := makeCertForTest(t, now.Add(-time.Hour), now.Add(90*24*time.Hour))
	address, _ := startTLSServerForTest(t, cert)

	job := tlsJobForIntegration(address, map[string]any{"verify_tls": false})

	result, repo, engine, lines := runPipeline(t, job)
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}

	if reachabilityState(engine, "ssl.reachability", 1) != health.HealthOK {
		t.Fatalf("expected ssl.reachability OK")
	}
	if repo.states["ssl.handshake_time_ms"] != health.HealthOK {
		t.Fatalf("expected ssl.handshake_time_ms OK, got %v", repo.states["ssl.handshake_time_ms"])
	}
	if repo.states["ssl.certificate_expiry_days"] != health.HealthOK {
		t.Fatalf("expected ssl.certificate_expiry_days OK, got %v", repo.states["ssl.certificate_expiry_days"])
	}

	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		`monitor_tls_reachability{monitor_id="monitor-1",monitor_type="tls",probe_location="amsterdam",resource_id="resource-1",workspace_id="workspace-1"} 1`,
		"monitor_tls_handshake_time_ms{",
		"monitor_tls_certificate_expiry_days{",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected line %q in:\n%s", want, joined)
		}
	}
}

func TestPipelineTLSExpiredFiresHealth(t *testing.T) {
	now := time.Now()
	cert := makeCertForTest(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	address, _ := startTLSServerForTest(t, cert)

	job := tlsJobForIntegration(address, map[string]any{"verify_tls": true})

	result, repo, engine, lines := runPipeline(t, job)
	if result.Success || result.ErrorCode != "certificate_expired" {
		t.Fatalf("expected certificate_expired, got %+v", result)
	}

	// Reachability is down and the (negative) expiry days trip the
	// LOWER_IS_WORSE critical threshold → HealthError, persisted.
	if reachabilityState(engine, "ssl.reachability", 0) != health.HealthError {
		t.Fatalf("expected ssl.reachability HealthError")
	}
	if repo.states["ssl.certificate_expiry_days"] != health.HealthError {
		t.Fatalf("expected ssl.certificate_expiry_days HealthError, got %v", repo.states["ssl.certificate_expiry_days"])
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, `monitor_tls_reachability{monitor_id="monitor-1",monitor_type="tls",probe_location="amsterdam",resource_id="resource-1",workspace_id="workspace-1"} 0`) {
		t.Fatalf("expected down reachability line in:\n%s", joined)
	}
}
