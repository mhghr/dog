package updater

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	agentupdater "monitoring-platform/internal/agent/updater"
)

// Updater periodically checks for and applies agent binary updates.
type Updater struct {
	currentVersion string
	channel        string
	binaryPath     string
	logger         *slog.Logger
	checker        *agentupdater.UpdateChecker
	checkNow       func(currentVersion, channel, osArch string) (*agentupdater.ReleaseManifest, error)
	apply          func(data []byte, path string) error
}

// NewUpdater creates an updater that targets binaryPath.
func NewUpdater(currentVersion, channel, baseURL, binaryPath string, logger *slog.Logger) *Updater {
	checker := agentupdater.NewUpdateChecker(baseURL)
	return &Updater{
		currentVersion: currentVersion,
		channel:        channel,
		binaryPath:     binaryPath,
		logger:         logger,
		checker:        checker,
		checkNow:       checker.CheckForUpdate,
		apply:          agentupdater.ApplyUpdate,
	}
}

// Start checks for updates every checkInterval until ctx is cancelled.
func (u *Updater) Start(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	u.check(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.check(ctx)
		}
	}
}

func (u *Updater) check(ctx context.Context) {
	osArch := platformKey()

	manifest, err := u.checkNow(u.currentVersion, u.channel, osArch)
	if err != nil {
		u.logger.Debug("update check failed", "error", err)
		return
	}
	if manifest == nil {
		return
	}

	artifact, ok := manifest.Artifacts[osArch]
	if !ok {
		u.logger.Warn("no artifact for platform", "os_arch", osArch)
		return
	}

	u.logger.Info("update available",
		"current", u.currentVersion,
		"available", manifest.Version,
	)

	data, err := agentupdater.DownloadArtifact(nil, artifact.URL)
	if err != nil {
		u.logger.Error("download update failed", "error", err)
		return
	}

	if _, err := agentupdater.VerifyArtifact(data, artifact.SHA256); err != nil {
		u.logger.Error("update checksum mismatch", "error", err)
		return
	}

	if err := u.apply(data, u.binaryPath); err != nil {
		u.logger.Error("apply update failed", "error", err)
		return
	}

	u.logger.Info("update applied, restart required", "version", manifest.Version)
}

func platformKey() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
