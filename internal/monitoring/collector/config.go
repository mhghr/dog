package collector

import (
	"fmt"
	"os"
	"os/exec"
)

// FindBinary locates the otelcol binary, checking common install paths then PATH.
func FindBinary() (string, error) {
	locations := []string{
		"/usr/bin/otelcol",
		"/usr/local/bin/otelcol",
		"/opt/monitoring-agent/bin/otelcol",
		"./otelcol",
	}
	for _, loc := range locations {
		if info, err := os.Stat(loc); err == nil && !info.IsDir() {
			return loc, nil
		}
	}
	if path, err := exec.LookPath("otelcol"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("otelcol binary not found")
}
