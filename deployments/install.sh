#!/bin/bash
set -euo pipefail

# Probe Agent installer
# Downloads the agent binary, verifies checksum, creates system user,
# installs systemd service, and runs enrollment.

RELEASE_REPO="${RELEASE_REPO:-mhghr/dog}"
CONTROL_PLANE="${CONTROL_PLANE:-https://api.example.com}"
AGENT_VERSION="${AGENT_VERSION:-v0.5.0}"
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

BINARY_NAME="probe-agent-${OS}-${ARCH}"
CHECKSUM_NAME="${BINARY_NAME}.sha256"

if [ "$AGENT_VERSION" = "latest" ]; then
    RELEASE_URL="https://github.com/${RELEASE_REPO}/releases/latest/download"
else
    RELEASE_URL="https://github.com/${RELEASE_REPO}/releases/download/${AGENT_VERSION}"
fi

BINARY_URL="${RELEASE_URL}/${BINARY_NAME}"
CHECKSUM_URL="${RELEASE_URL}/${CHECKSUM_NAME}"

echo "Downloading probe-agent ${AGENT_VERSION} for ${OS}/${ARCH}..."
echo "  Binary: ${BINARY_URL}"
curl -fsSL "$BINARY_URL" -o "/tmp/${BINARY_NAME}"
curl -fsSL "$CHECKSUM_URL" -o "/tmp/${CHECKSUM_NAME}"

echo "Verifying checksum..."
cd /tmp
echo "$(cat ${CHECKSUM_NAME})  ${BINARY_NAME}" | sha256sum -c -
chmod +x "/tmp/${BINARY_NAME}"

if ! id -u probe-agent >/dev/null 2>&1; then
    echo "Creating probe-agent user..."
    useradd --system --no-create-home --shell /usr/sbin/nologin probe-agent
fi

mkdir -p "$INSTALL_DIR" "$STATE_DIR" "$CONFIG_DIR" "$LOG_DIR"
chown probe-agent:probe-agent "$STATE_DIR" "$CONFIG_DIR" "$LOG_DIR"
chmod 750 "$STATE_DIR" "$CONFIG_DIR"

mv "/tmp/${BINARY_NAME}" "$INSTALL_DIR/probe-agent"
setcap cap_net_raw=ep "$INSTALL_DIR/probe-agent"

GATEWAY_HOST=$(echo "$CONTROL_PLANE" | sed -e 's|^https\?://||' -e 's|/.*||')

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    cat > "$CONFIG_DIR/config.yaml" << EOF
control_plane: ${CONTROL_PLANE}
agent_gateway: ${GATEWAY_HOST}:8443
state_dir: ${STATE_DIR}
log_level: info
enrollment_token: ${ENROLLMENT_TOKEN}
worker_concurrency: 200
spool_dir: ${STATE_DIR}/spool
ping_privileged: true
EOF
    chmod 640 "$CONFIG_DIR/config.yaml"
    chown probe-agent:probe-agent "$CONFIG_DIR/config.yaml"
fi

if [ -f "$(dirname "$0")/probe-agent.service" ]; then
    cp "$(dirname "$0")/probe-agent.service" /etc/systemd/system/probe-agent.service
elif [ -f "/tmp/probe-agent.service" ]; then
    cp /tmp/probe-agent.service /etc/systemd/system/probe-agent.service
else
    cat > /etc/systemd/system/probe-agent.service << 'UNIT'
[Unit]
Description=Probe Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=probe-agent
Group=probe-agent
ExecStart=/usr/local/bin/probe-agent run
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/probe-agent /var/log/probe-agent
CapabilityBoundingSet=CAP_NET_RAW
AmbientCapabilities=CAP_NET_RAW
EnvironmentFile=-/etc/probe-agent/config.env

[Install]
WantedBy=multi-user.target
UNIT
fi

systemctl daemon-reload

echo ""
echo "=== Installation complete ==="
echo ""
echo "Next steps:"
echo "1. Set ENROLLMENT_TOKEN in $CONFIG_DIR/config.yaml"
echo "2. Run: sudo -u probe-agent /usr/local/bin/probe-agent enroll"
echo "3. Have an admin approve the agent in the console"
echo "4. Run: sudo systemctl enable --now probe-agent"
echo "5. Check status: sudo systemctl status probe-agent"
