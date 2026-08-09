package updater

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ArtifactRef struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type ReleaseManifest struct {
	Version         string                 `json:"version"`
	Channel         string                 `json:"channel"`
	MinimumProtocol int                    `json:"minimum_protocol"`
	MaximumProtocol int                    `json:"maximum_protocol"`
	PublishedAt     string                 `json:"published_at,omitempty"`
	Artifacts       map[string]ArtifactRef `json:"artifacts"`
}

type UpdateChecker struct {
	baseURL    string
	httpClient *http.Client
}

func NewUpdateChecker(baseURL string) *UpdateChecker {
	return &UpdateChecker{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *UpdateChecker) CheckForUpdate(currentVersion, channel, osArch string) (*ReleaseManifest, error) {
	url := fmt.Sprintf("%s/api/agent/updates/release-manifest.json", c.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var manifest ReleaseManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	if manifest.Channel != channel {
		return nil, fmt.Errorf("channel mismatch: expected %s, got %s", channel, manifest.Channel)
	}

	if manifest.Version == currentVersion {
		return nil, nil
	}

	artifact, ok := manifest.Artifacts[osArch]
	if !ok {
		return nil, fmt.Errorf("no artifact for %s", osArch)
	}
	if artifact.URL == "" {
		return nil, fmt.Errorf("empty artifact URL for %s", osArch)
	}
	if artifact.SHA256 == "" {
		return nil, fmt.Errorf("missing checksum for %s", osArch)
	}

	return &manifest, nil
}

func VerifyArtifact(data []byte, expectedSHA256 string) ([]byte, error) {
	hash := sha256.Sum256(data)
	actual := hex.EncodeToString(hash[:])

	if actual != expectedSHA256 {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actual)
	}

	return data, nil
}

func DownloadArtifact(httpClient *http.Client, url string) ([]byte, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	buf := &bytes.Buffer{}
	limited := io.LimitReader(resp.Body, 256*1024*1024)

	if _, err := io.Copy(buf, limited); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return buf.Bytes(), nil
}

func ApplyUpdate(binary []byte, targetPath string) error {
	if len(binary) == 0 {
		return fmt.Errorf("empty binary")
	}

	tmpPath := targetPath + ".new"

	if err := os.WriteFile(tmpPath, binary, 0o755); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}
