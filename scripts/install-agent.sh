#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-$HOME/probe-agent}"
REPO="mhghr/dog"

if [ -z "$TOKEN" ]; then
    echo "Usage:"
    echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/scripts/install-agent.sh | bash -s -- <TOKEN> [CONTROL_PLANE_URL]"
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
echo ""

detect_arch() {
    local arch
    case "$(uname -s)" in
        Linux)
            case "$(uname -m)" in
                x86_64|amd64) arch="linux-amd64" ;;
                aarch64|arm64) arch="linux-arm64" ;;
                *) arch="linux-$(uname -m)" ;;
            esac
            ;;
        Darwin)
            case "$(uname -m)" in
                x86_64|amd64) arch="darwin-amd64" ;;
                arm64) arch="darwin-arm64" ;;
                *) arch="darwin-$(uname -m)" ;;
            esac
            ;;
        *) arch="$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)" ;;
    esac
    echo "$arch"
}

download_binary() {
    local arch="$1"
    local dest="$2"
    local url="https://github.com/$REPO/releases/latest/download/probe-agent-${arch}"

    echo "   Downloading $url ..."
    if curl -fsSL -L --retry 3 -o "$dest" "$url"; then
        chmod +x "$dest"
        return 0
    fi

    url="https://github.com/$REPO/releases/download/latest/probe-agent-${arch}"
    if curl -fsSL -L --retry 3 -o "$dest" "$url"; then
        chmod +x "$dest"
        return 0
    fi

    return 1
}

build_from_source() {
    echo "   Building from source..."
    TMP=$(mktemp -d)
    trap "rm -rf $TMP" RETURN

    local archive_url="https://github.com/$REPO/archive/refs/heads/main.tar.gz"

    echo "   Downloading source archive..."
    if ! curl -fsSL -L --retry 3 "$archive_url" -o "$TMP/source.tar.gz"; then
        echo "   Falling back to git clone..."
        if ! command -v git &>/dev/null; then
            echo "   [ERROR] git is not available and archive download failed."
            return 1
        fi
        git clone --depth 1 "https://github.com/$REPO.git" "$TMP/src"
        mv "$TMP/src"/* "$TMP/src"/.[!.]* "$TMP/" 2>/dev/null || true
    else
        tar -xzf "$TMP/source.tar.gz" -C "$TMP"
        local extracted_dir
        extracted_dir=$(ls -d "$TMP"/dog-* 2>/dev/null || ls -d "$TMP"/*/ 2>/dev/null | head -1)
        if [ -n "$extracted_dir" ]; then
            shopt -s dotglob
            mv "$extracted_dir"/* "$TMP/" 2>/dev/null || true
            rmdir "$extracted_dir" 2>/dev/null || true
            shopt -u dotglob
        fi
    fi

    if ! command -v go &>/dev/null; then
        echo "   [ERROR] Go is not installed."
        echo "   Install Go: sudo apt install golang-go"
        return 1
    fi

    cd "$TMP"
    echo "   Compiling..."
    go build -ldflags="-s -w" -o "$AGENT_DIR/probe-agent" ./cmd/probe-agent
    chmod +x "$AGENT_DIR/probe-agent"
}

ARCH=$(detect_arch)
echo "[1/3] Downloading probe-agent ($ARCH)..."

mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"

if download_binary "$ARCH" "$AGENT_DIR/probe-agent"; then
    echo "   Downloaded pre-built binary"
else
    echo "   No pre-built binary found for $ARCH"
    build_from_source || exit 1
fi

echo "   Binary: $AGENT_DIR/probe-agent"

echo "[2/3] Writing config..."
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

echo "[3/3] Starting probe-agent..."
echo "==================================="
export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
exec "$AGENT_DIR/probe-agent" run
