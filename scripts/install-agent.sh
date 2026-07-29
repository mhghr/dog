#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-$HOME/probe-agent}"
REPO="mhghr/dog"
GH_TOKEN="${GITHUB_TOKEN:-}"

SERVICE_FILE="/etc/systemd/system/probe-agent.service"
USE_SYSTEMD=false
if [ -d /run/systemd/system ]; then
    USE_SYSTEMD=true
fi

if [ -z "$TOKEN" ]; then
    echo "Usage:"
    echo "  ./install-agent.sh <ENROLLMENT_TOKEN> [CONTROL_PLANE_URL]"
    echo "  AGENT_ENROLLMENT_TOKEN=xxx ./install-agent.sh"
    exit 1
fi

case "$(uname -m)" in
    x86_64|amd64) ARCH="linux-amd64" ;;
    aarch64|arm64) ARCH="linux-arm64" ;;
    *) echo "[ERROR] Unsupported arch: $(uname -m)"; exit 1 ;;
esac

echo "==================================="
echo " probe-agent installer"
echo "==================================="
echo " Control plane : $CONTROL_PLANE"
echo " Arch          : $ARCH"
echo " Install method: $([ "$USE_SYSTEMD" = true ] && echo 'systemd (persistent)' || echo 'background process')"
echo "==================================="

# Download binary
BIN_URL="https://github.com/$REPO/releases/latest/download/probe-agent-${ARCH}"
AUTH_ARGS=()
if [ -n "$GH_TOKEN" ]; then
    AUTH_ARGS=(-H "Authorization: Bearer $GH_TOKEN")
fi

echo "Downloading $BIN_URL ..."
mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"

if ! curl -fsSL -L --retry 3 \
    "${AUTH_ARGS[@]}" \
    -H "Accept: application/octet-stream" \
    -o "$AGENT_DIR/probe-agent" \
    "$BIN_URL"; then
    echo "[ERROR] Download failed."
    exit 1
fi

chmod +x "$AGENT_DIR/probe-agent"
echo "Binary: $AGENT_DIR/probe-agent ($(du -h "$AGENT_DIR/probe-agent" | cut -f1))"

echo "Writing config..."
cat > "$AGENT_DIR/config.yaml" <<YAML
control_plane: "$CONTROL_PLANE"
agent_gateway: "${AGENT_GATEWAY:-localhost:8443}"
enrollment_token: "$TOKEN"
state_dir: "$AGENT_DIR/state"
spool_dir: "$AGENT_DIR/spool"
health_address: ":8081"
log_level: "info"
log_format: "json"
YAML

if [ "$USE_SYSTEMD" = true ] && [ "$(id -u)" = "0" ]; then
    echo "Installing systemd service..."
    cat > "$SERVICE_FILE" <<UNIT
[Unit]
Description=Probe Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=$AGENT_DIR/probe-agent run
Restart=always
RestartSec=5
LimitNOFILE=65536
WorkingDirectory=$AGENT_DIR
Environment=AGENT_CONFIG_PATH=$AGENT_DIR/config.yaml
CapabilityBoundingSet=CAP_NET_RAW
AmbientCapabilities=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
UNIT

    systemctl daemon-reload
    systemctl enable probe-agent
    systemctl restart probe-agent
    echo "   systemd service installed and started."
    echo "   It will auto-start on boot and restart on failure."
    echo "   Manage with: systemctl [start|stop|status] probe-agent"
elif [ "$USE_SYSTEMD" = true ] && [ "$(id -u)" != "0" ]; then
    echo "   Not running as root, skipping systemd install."
    echo "   Starting in background instead..."
    export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
    nohup "$AGENT_DIR/probe-agent" run > "$AGENT_DIR/agent.log" 2>&1 &
    PID=$!
    echo "   Agent started in background (PID: $PID)."
    echo "   Logs: $AGENT_DIR/agent.log"
    echo ""
    echo "   To install as systemd service run:"
    echo "     sudo ./install-agent.sh $TOKEN $CONTROL_PLANE"
    echo "   or manually:"
    echo "     sudo cp deployments/probe-agent.service $SERVICE_FILE"
    echo "     sudo systemctl enable --now probe-agent"
else
    echo "Starting in background..."
    export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
    nohup "$AGENT_DIR/probe-agent" run > "$AGENT_DIR/agent.log" 2>&1 &
    PID=$!
    echo "   Agent started in background (PID: $PID)."
    echo "   Logs: $AGENT_DIR/agent.log"
    echo "   To stop: kill $PID"
fi
