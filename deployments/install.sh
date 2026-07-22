#!/bin/bash
set -euo pipefail

# Probe Agent installer
# Downloads the agent binary, verifies checksum, creates system user,
# installs systemd service, and runs enrollment.

CONTROL_PLANE="${CONTROL_PLANE:-https://control.example.com}"
AGENT_VERSION="${AGENT_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
STATE_DIR="${STATE_DIR:-/var/lib/probe-agent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/probe-agent}"
LOG_DIR="${LOG_DIR:-/var/log/probe-agent}"
ENROLLMENT_TOKEN="${ENROLLMENT_TOKEN:-}"

echo "=== Probe Agent Installer ==="

OS="linux"
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY_URL="${CONTROL_PLANE}/releases/download/${AGENT_VERSION}/probe-agent-${OS}-${ARCH}"
CHECKSUM_URL="${BINARY_URL}.sha256"

echo "Downloading probe-agent ${AGENT_VERSION} for ${OS}/${ARCH}..."
curl -fsSL "$BINARY_URL" -o /tmp/probe-agent
curl -fsSL "$CHECKSUM_URL" -o /tmp/probe-agent.sha256

echo "Verifying checksum..."
cd /tmp
sha256sum -c probe-agent.sha256
chmod +x /tmp/probe-agent

if ! id -u probe-agent >/dev/null 2>&1; then
    echo "Creating probe-agent user..."
    useradd --system --no-create-home --shell /usr/sbin/nologin probe-agent
fi

mkdir -p "$INSTALL_DIR" "$STATE_DIR" "$CONFIG_DIR" "$LOG_DIR"
chown probe-agent:probe-agent "$STATE_DIR" "$CONFIG_DIR" "$LOG_DIR"
chmod 750 "$STATE_DIR" "$CONFIG_DIR"

mv /tmp/probe-agent "$INSTALL_DIR/probe-agent"
setcap cap_net_raw=ep "$INSTALL_DIR/probe-agent"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    cat > "$CONFIG_DIR/config.yaml" << EOF
control_plane: ${CONTROL_PLANE}
agent_gateway: ${CONTROL_PLANE}:443
state_dir: ${STATE_DIR}
log_level: info
enrollment_token: ${ENROLLMENT_TOKEN}
runtime:
  max_concurrency: 200
  shutdown_grace_period: 30s
spool:
  max_bytes: 1073741824
  max_results: 100000
  flush_interval: 250ms
  batch_size: 250
updates:
  channel: stable
  auto_update: false
EOF
    chmod 640 "$CONFIG_DIR/config.yaml"
    chown probe-agent:probe-agent "$CONFIG_DIR/config.yaml"
fi

cp deployments/probe-agent.service /etc/systemd/system/probe-agent.service
systemctl daemon-reload

echo ""
echo "=== Installation complete ==="
echo ""
echo "Next steps:"
echo "1. Set ENROLLMENT_TOKEN in $CONFIG_DIR/config.yaml"
echo "2. Run: sudo -u probe-agent probe-agent enroll"
echo "3. Have an admin approve the agent in the console"
echo "4. Run: sudo systemctl enable --now probe-agent"
echo "5. Check status: sudo systemctl status probe-agent"
