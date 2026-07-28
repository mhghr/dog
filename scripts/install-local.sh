#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-$HOME/probe-agent}"

if [ -z "$TOKEN" ]; then
    echo "Usage: AGENT_ENROLLMENT_TOKEN=<TOKEN> bash install-local.sh [CONTROL_PLANE_URL]"
    exit 1
fi

SOURCE_DIR="${PROBE_SOURCE:-.}"

echo "==================================="
echo " probe-agent builder"
echo "==================================="
echo " Control plane : $CONTROL_PLANE"
echo " Install dir   : $AGENT_DIR"
echo " Source dir    : $SOURCE_DIR"
echo "==================================="

if [ ! -f "$SOURCE_DIR/go.mod" ]; then
    echo "[ERROR] No go.mod found in $SOURCE_DIR. Run this from the repo root or set PROBE_SOURCE."
    exit 1
fi

if ! command -v go &>/dev/null; then
    echo "[ERROR] Go is not installed. Run: sudo apt install golang-go"
    exit 1
fi

echo "Building..."
mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"
cd "$SOURCE_DIR"
go build -ldflags="-s -w" -o "$AGENT_DIR/probe-agent" ./cmd/probe-agent

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

echo "Starting..."
export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
exec "$AGENT_DIR/probe-agent" run
