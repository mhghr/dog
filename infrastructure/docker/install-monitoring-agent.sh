#!/usr/bin/env bash
set -euo pipefail

# Monitoring Agent installer
# Usage: curl -fsSL https://agent.example.com/install.sh | sudo bash
#
# Required env: MONITORING_BOOTSTRAP_TOKEN (one-time token from the dashboard)
# Optional env:
#   MONITORING_SERVER_URL    (default: https://monitor.example.com)
#   MONITORING_OTEL_ENDPOINT (default: https://monitor.example.com:4318)
#   MONITORING_VERSION       (default: latest)
#   MONITORING_STATE_DIR     (default: /var/lib/monitoring-agent)

MONITORING_VERSION="${MONITORING_VERSION:-latest}"
MONITORING_SERVER_URL="${MONITORING_SERVER_URL:-https://monitor.example.com}"
MONITORING_OTEL_ENDPOINT="${MONITORING_OTEL_ENDPOINT:-https://monitor.example.com:4318}"
MONITORING_STATE_DIR="${MONITORING_STATE_DIR:-/var/lib/monitoring-agent}"

if [ -z "${MONITORING_BOOTSTRAP_TOKEN:-}" ]; then
    echo "Error: MONITORING_BOOTSTRAP_TOKEN is required" >&2
    echo ""
    echo "Usage: curl -fsSL https://agent.example.com/install.sh | sudo bash" >&2
    echo ""
    echo "Environment variables:" >&2
    echo "  MONITORING_BOOTSTRAP_TOKEN  (required) one-time token from the dashboard" >&2
    echo "  MONITORING_SERVER_URL       (optional) default: https://monitor.example.com" >&2
    echo "  MONITORING_OTEL_ENDPOINT    (optional) default: https://monitor.example.com:4318" >&2
    echo "  MONITORING_VERSION          (optional) default: latest" >&2
    exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  PLATFORM="linux/amd64" ;;
    aarch64) PLATFORM="linux/arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

DOWNLOAD_BASE="${MONITORING_DOWNLOAD_BASE:-https://agent.example.com/releases/${MONITORING_VERSION}}"
BINARY_URL="${DOWNLOAD_BASE}/monitoring-agent-${PLATFORM}"
CHECKSUM_URL="${BINARY_URL}.sha256"

echo "==> Downloading monitoring-agent ${MONITORING_VERSION} (${PLATFORM})"
curl -fsSL "$BINARY_URL" -o /tmp/monitoring-agent
curl -fsSL "$CHECKSUM_URL" -o /tmp/monitoring-agent.sha256

echo "==> Verifying checksum"
( cd /tmp && echo "$(awk '{print $1}' monitoring-agent.sha256)  monitoring-agent" | sha256sum -c - )

echo "==> Installing binary"
install -m 0755 /tmp/monitoring-agent /usr/bin/monitoring-agent

echo "==> Creating state directory"
mkdir -p "${MONITORING_STATE_DIR}"
chown nobody:nogroup "${MONITORING_STATE_DIR}"
chmod 0700 "${MONITORING_STATE_DIR}"

echo "==> Writing environment configuration"
ENV_FILE="/etc/monitoring-agent/env"
mkdir -p "$(dirname "$ENV_FILE")"
cat > "$ENV_FILE" << EOF
MONITORING_SERVER_URL=${MONITORING_SERVER_URL}
MONITORING_OTEL_ENDPOINT=${MONITORING_OTEL_ENDPOINT}
MONITORING_BOOTSTRAP_TOKEN=${MONITORING_BOOTSTRAP_TOKEN}
MONITORING_STATE_DIR=${MONITORING_STATE_DIR}
MONITORING_LOG_LEVEL=info
MONITORING_LOG_FORMAT=json
EOF
chmod 0600 "$ENV_FILE"

echo "==> Installing systemd service"
cat > /etc/systemd/system/monitoring-agent.service << 'SERVICE'
[Unit]
Description=Monitoring Agent
Documentation=https://docs.example.com/monitoring-agent
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=10min
StartLimitBurst=5

[Service]
Type=simple
User=nobody
Group=nogroup
EnvironmentFile=/etc/monitoring-agent/env
ExecStart=/usr/bin/monitoring-agent
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/monitoring-agent
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable monitoring-agent
systemctl restart monitoring-agent

echo ""
echo "monitoring-agent installed and started"
echo ""
echo "Check status:  systemctl status monitoring-agent"
echo "Check logs:    journalctl -u monitoring-agent -f"
