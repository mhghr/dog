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
echo ""

install_go_pkg() {
    if command -v apt-get &>/dev/null; then
        apt-get update -qq && apt-get install -y -qq golang-go
    elif command -v dnf &>/dev/null; then
        dnf install -y -q golang
    elif command -v yum &>/dev/null; then
        yum install -y -q golang
    elif command -v apk &>/dev/null; then
        apk add --no-cache go
    elif command -v pacman &>/dev/null; then
        pacman -S --noconfirm go
    elif command -v zypper &>/dev/null; then
        zypper install -y go
    else
        return 1
    fi
}

install_git_pkg() {
    if command -v apt-get &>/dev/null; then
        apt-get update -qq && apt-get install -y -qq git
    elif command -v dnf &>/dev/null; then
        dnf install -y -q git
    elif command -v yum &>/dev/null; then
        yum install -y -q git
    elif command -v apk &>/dev/null; then
        apk add --no-cache git
    elif command -v pacman &>/dev/null; then
        pacman -S --noconfirm git
    elif command -v zypper &>/dev/null; then
        zypper install -y git
    else
        return 1
    fi
}

need_go=false
need_git=false

if ! command -v go &>/dev/null; then need_go=true; fi
if ! command -v git &>/dev/null; then need_git=true; fi

if $need_go; then
    echo "[1/5] Installing Go..."
    if install_go_pkg; then
        echo "   Go $(go version) installed via package manager"
    else
        echo "   Package manager not available, downloading Go binary..."
        GO_ARCH="linux-amd64"
        case "$(uname -m)" in
            aarch64|arm64) GO_ARCH="linux-arm64" ;;
        esac
        GO_URL="https://go.dev/dl/go1.22.10.${GO_ARCH}.tar.gz"
        echo "   Downloading ${GO_URL}..."
        curl -fsSL -L --retry 3 "$GO_URL" -o "/tmp/go.tar.gz"
        mkdir -p "$HOME/.go"
        tar -C "$HOME/.go" -xzf "/tmp/go.tar.gz"
        rm -f "/tmp/go.tar.gz"
        export GOROOT="$HOME/.go/go"
        export PATH="$GOROOT/bin:$PATH"
        echo "   Go $(go version) installed"
    fi
else
    echo "[1/5] Go: $(go version)"
fi

if $need_git; then
    echo "   Installing git..."
    if ! install_git_pkg; then
        echo "   [ERROR] Cannot install git. Install git manually and re-run."
        exit 1
    fi
    echo "   git installed"
fi

echo "[2/5] Creating directories..."
mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"

echo "[3/5] Building probe-agent..."
TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

git clone --depth 1 https://github.com/mhghr/dog.git "$TMP"
cd "$TMP"
go build -ldflags="-s -w" -o "$AGENT_DIR/probe-agent" ./cmd/probe-agent
echo "   Binary: $AGENT_DIR/probe-agent"

echo "[4/5] Writing config..."
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

echo "[5/5] Starting probe-agent..."
echo "==================================="
export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
exec "$AGENT_DIR/probe-agent" run
