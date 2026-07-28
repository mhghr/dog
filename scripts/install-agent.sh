#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-$HOME/probe-agent}"
REPO="mhghr/dog"
GH_TOKEN="${GITHUB_TOKEN:-}"

if [ -z "$TOKEN" ]; then
    echo "Usage:"
    echo "  ./install-agent.sh <ENROLLMENT_TOKEN> [CONTROL_PLANE_URL]"
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
echo "==================================="

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

echo "Starting probe-agent..."
export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
exec "$AGENT_DIR/probe-agent" run
