#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-$HOME/probe-agent}"
REPO="mhghr/dog"
BIN="$AGENT_DIR/probe-agent"

if [ -z "$TOKEN" ]; then
    echo "Usage:"
    echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/scripts/install-agent.sh | bash -s -- <TOKEN> [CONTROL_PLANE]"
    echo "  AGENT_ENROLLMENT_TOKEN=xxx bash install-agent.sh"
    exit 1
fi

echo "==================================="
echo " probe-agent installer"
echo "==================================="
echo " Control plane : $CONTROL_PLANE"
echo " Install dir   : $AGENT_DIR"
echo "==================================="

# --- Install Go if missing ---
install_go() {
    echo "[1/3] Installing Go..."
    case "$(uname -m)" in
        x86_64|amd64) GOARCH="amd64" ;;
        aarch64|arm64) GOARCH="arm64" ;;
        *) echo "Unsupported arch: $(uname -m)"; exit 1 ;;
    esac
    local GO_VER=$(curl -fsSL https://go.dev/VERSION?m=text | head -1)
    [ -z "$GO_VER" ] && GO_VER="go1.24.0"
    local TAR="${GO_VER}.linux-${GOARCH}.tar.gz"
    curl -fsSL -o /tmp/$TAR "https://go.dev/dl/$TAR"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/$TAR
    rm /tmp/$TAR
    export PATH="/usr/local/go/bin:$PATH"
    echo "   Go $(go version) installed"
}

if ! command -v go &> /dev/null; then
    install_go
else
    echo "[1/3] Go: $(go version)"
fi

# --- Build probe-agent ---
echo "[2/3] Building probe-agent..."
mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"

# If run from inside the repo, build locally
SCRIPT_DIR="$(cd "$(dirname "$0")" 2>/dev/null && pwd || echo "")"
if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/../../cmd/probe-agent/main.go" ]; then
    cd "$SCRIPT_DIR/../.."
    go build -ldflags="-s -w" -o "$BIN" ./cmd/probe-agent
elif [ -f "./cmd/probe-agent/main.go" ]; then
    go build -ldflags="-s -w" -o "$BIN" ./cmd/probe-agent
else
    TMP_DIR=$(mktemp -d)
    echo "   Cloning $REPO..."
    git clone --depth 1 "https://github.com/$REPO.git" "$TMP_DIR" 2>/dev/null || {
        echo "   Git not available, downloading source tarball..."
        curl -fsSL -o /tmp/repo.tar.gz "https://github.com/$REPO/archive/refs/heads/main.tar.gz"
        tar -xzf /tmp/repo.tar.gz -C "$TMP_DIR" --strip-components=1
        rm /tmp/repo.tar.gz
    }
    cd "$TMP_DIR"
    go build -ldflags="-s -w" -o "$BIN" ./cmd/probe-agent
    rm -rf "$TMP_DIR"
fi
echo "   Binary: $BIN ($(du -h "$BIN" | cut -f1))"

# --- Write config ---
echo "[3/3] Writing config..."
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

# --- Install as systemd service (if root) or run in background ---
if [ -d /run/systemd/system ] && [ "$(id -u)" = "0" ]; then
    echo "   Installing systemd service..."
    cat > /etc/systemd/system/probe-agent.service <<UNIT
[Unit]
Description=Probe Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=$BIN run
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
    echo "   Service installed and started (auto-start on boot, auto-restart on failure)."
    echo "   Manage: systemctl [start|stop|status] probe-agent"
else
    echo "   Starting in background (not root or no systemd)..."
    export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
    nohup "$BIN" run > "$AGENT_DIR/agent.log" 2>&1 &
    echo "   PID: $!"
    echo "   Logs: $AGENT_DIR/agent.log"
    echo "   To stop: kill $!"
fi

echo "==================================="
echo " Done. Agent will auto-enroll on first run and connect to gateway."
echo " Check status: journalctl -u probe-agent -f  (systemd)"
echo "           or: tail -f $AGENT_DIR/agent.log  (background)"
echo "==================================="
