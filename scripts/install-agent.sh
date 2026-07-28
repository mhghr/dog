#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-/opt/probe-agent}"

if [ -z "$TOKEN" ]; then
    echo "Usage:"
    echo "  curl -fsSL https://raw.githubusercontent.com/mhghr/dog/main/scripts/install-agent.sh | bash -s -- <TOKEN> [CONTROL_PLANE_URL]"
    echo "  or"
    echo "  AGENT_ENROLLMENT_TOKEN=<TOKEN> bash install-agent.sh"
    exit 1
fi

echo "==================================="
echo " probe-agent installer"
echo "==================================="
echo " Control plane : $CONTROL_PLANE"
echo " Install dir   : $AGENT_DIR"
echo "==================================="

if ! command -v go &>/dev/null; then
    echo "[ERROR] Go is not installed. Install from https://go.dev/dl/"
    echo "  Linux:   sudo snap install go --classic"
    echo "  macOS:   brew install go"
    exit 1
fi

echo "[1/4] Creating directories..."
mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"

if [ ! -f "$AGENT_DIR/probe-agent" ]; then
    echo "[2/4] Building probe-agent..."
    TMP=$(mktemp -d)
    trap "rm -rf $TMP" EXIT

    git clone --depth 1 https://github.com/mhghr/dog.git "$TMP"
    cd "$TMP"
    go build -ldflags="-s -w" -o "$AGENT_DIR/probe-agent" ./cmd/probe-agent
    echo "   Binary installed to $AGENT_DIR/probe-agent"
else
    echo "[2/4] Binary already exists, skipping build"
fi

echo "[3/4] Writing config..."
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
echo "   Config: $AGENT_DIR/config.yaml"

echo "[4/4] Starting probe-agent..."
export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
exec "$AGENT_DIR/probe-agent" run
