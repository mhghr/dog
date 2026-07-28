package enrollment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"monitoring-platform/internal/agent"
)

type EnrollRequest struct {
	EnrollmentToken    string   `json:"enrollment_token"`
	Hostname           string   `json:"hostname"`
	MachineFingerprint string   `json:"machine_fingerprint"`
	PublicKey          string   `json:"public_key"`
	Version            string   `json:"version"`
	OperatingSystem    string   `json:"operating_system"`
	Architecture       string   `json:"architecture"`
	PrivateIPs         []string `json:"private_ips"`
	Capabilities       []string `json:"capabilities"`
	MaxConcurrency     int32    `json:"max_concurrency"`
}

type EnrollResponse struct {
	AgentID     string `json:"agent_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	AgentSecret string `json:"agent_secret"`
}

type StatusResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	LocationID        string `json:"location_id"`
	CertificateSerial string `json:"certificate_serial,omitempty"`
	Certificate       string `json:"certificate,omitempty"`
	ApprovedAt        string `json:"approved_at,omitempty"`
}

type EnrollResult struct {
	AgentID     string
	Certificate string
	PrivateKey  string
}

func Enroll(ctx context.Context, cfg agent.AgentConfig, version string, logger *slog.Logger) (*EnrollResult, error) {
	if cfg.EnrollmentToken == "" {
		return nil, fmt.Errorf("enrollment token is required")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	privateIPs := getPrivateIPs()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}))

	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}))

	fingerprint := machineFingerprint(hostname, publicKeyPEM)

	req := EnrollRequest{
		EnrollmentToken:    cfg.EnrollmentToken,
		Hostname:           hostname,
		MachineFingerprint: fingerprint,
		PublicKey:          publicKeyPEM,
		Version:            version,
		OperatingSystem:    runtime.GOOS,
		Architecture:       runtime.GOARCH,
		PrivateIPs:         privateIPs,
		Capabilities:       []string{"http", "https", "dns", "tcp", "icmp", "tls"},
		MaxConcurrency:     int32(cfg.WorkerConcurrency),
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal enroll request: %w", err)
	}

	controlPlane := strings.TrimRight(cfg.ControlPlane, "/")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, controlPlane+"/api/v1/agent/v1/enroll", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create enroll request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("enroll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("enrollment API returned %s", resp.Status)
	}

	var enrollResp EnrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return nil, fmt.Errorf("decode enroll response: %w", err)
	}

	logger.Info("enrollment request submitted", "agent_id", enrollResp.AgentID, "status", enrollResp.Status)

	if enrollResp.Status == "approved" {
		return &EnrollResult{
			AgentID:    enrollResp.AgentID,
			Certificate: "",
			PrivateKey: privateKeyPEM,
		}, nil
	}

	pollInterval := 5 * time.Second
	pollTimeout := 5 * time.Minute
	deadline := time.Now().Add(pollTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		statusResp, err := getStatus(ctx, controlPlane, enrollResp.AgentID, enrollResp.AgentSecret)
		if err != nil {
			logger.Warn("failed to check enrollment status", "error", err)
			continue
		}

		logger.Info("enrollment status", "agent_id", statusResp.ID, "status", statusResp.Status)

		if statusResp.Status == "approved" {
			return &EnrollResult{
				AgentID:     enrollResp.AgentID,
				Certificate: statusResp.Certificate,
				PrivateKey:  privateKeyPEM,
			}, nil
		}

		if statusResp.Status == "rejected" {
			return nil, fmt.Errorf("enrollment was rejected")
		}
	}

	return nil, fmt.Errorf("enrollment approval timed out after %s", pollTimeout)
}

func getStatus(ctx context.Context, controlPlane, agentID, secret string) (*StatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/agent/v1/status/%s?secret=%s", controlPlane, agentID, secret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status API returned %s", resp.Status)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

func machineFingerprint(hostname, publicKey string) string {
	h := sha256.New()
	h.Write([]byte(hostname))
	h.Write([]byte(publicKey))
	return hex.EncodeToString(h.Sum(nil))
}

func getPrivateIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.IsPrivate() {
			ips = append(ips, ipnet.IP.String())
		}
	}
	return ips
}
