#!/bin/bash
set -e

TOKEN="${1:-$AGENT_ENROLLMENT_TOKEN}"
CONTROL_PLANE="${2:-${AGENT_CONTROL_PLANE:-http://localhost:5000}}"
AGENT_DIR="${AGENT_DIR:-$HOME/probe-agent}"

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

install_go() {
    if command -v go &>/dev/null; then
        echo "   Go: $(go version)"
        return 0
    fi

    echo "   Installing Go..."
    local use_sudo=""
    if [ "$(id -u)" != "0" ] && command -v sudo &>/dev/null; then
        use_sudo="sudo"
    fi

    if command -v apt-get &>/dev/null; then
        $use_sudo apt-get update -qq && $use_sudo apt-get install -y -qq golang-go && return 0
    elif command -v dnf &>/dev/null; then
        $use_sudo dnf install -y -q golang && return 0
    elif command -v yum &>/dev/null; then
        $use_sudo yum install -y -q golang && return 0
    elif command -v apk &>/dev/null; then
        $use_sudo apk add --no-cache go && return 0
    elif command -v pacman &>/dev/null; then
        $use_sudo pacman -S --noconfirm go && return 0
    elif command -v zypper &>/dev/null; then
        $use_sudo zypper install -y go && return 0
    elif command -v snap &>/dev/null; then
        $use_sudo snap install go --classic && return 0
    fi
    return 1
}

install_git() {
    if command -v git &>/dev/null; then
        return 0
    fi

    local use_sudo=""
    if [ "$(id -u)" != "0" ] && command -v sudo &>/dev/null; then
        use_sudo="sudo"
    fi

    if command -v apt-get &>/dev/null; then
        $use_sudo apt-get update -qq && $use_sudo apt-get install -y -qq git && return 0
    elif command -v dnf &>/dev/null; then
        $use_sudo dnf install -y -q git && return 0
    elif command -v yum &>/dev/null; then
        $use_sudo yum install -y -q git && return 0
    elif command -v apk &>/dev/null; then
        $use_sudo apk add --no-cache git && return 0
    elif command -v pacman &>/dev/null; then
        $use_sudo pacman -S --noconfirm git && return 0
    elif command -v zypper &>/dev/null; then
        $use_sudo zypper install -y git && return 0
    fi
    return 1
}

if ! install_go; then
    echo ""
    echo "[ERROR] Could not install Go automatically."
    echo "Install Go manually then re-run this script:"
    echo "  sudo apt install golang-go    # Ubuntu/Debian"
    echo "  sudo dnf install golang       # Fedora"
    echo "  https://go.dev/dl/            # Manual download"
    exit 1
fi

if ! install_git; then
    echo ""
    echo "[ERROR] Could not install git automatically."
    echo "Install git manually: sudo apt install git"
    exit 1
fi

echo "[2/4] Creating directories..."
mkdir -p "$AGENT_DIR/state" "$AGENT_DIR/spool"

echo "[3/4] Building probe-agent..."
TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

git clone --depth 1 https://github.com/mhghr/dog.git "$TMP"
cd "$TMP"
go build -ldflags="-s -w" -o "$AGENT_DIR/probe-agent" ./cmd/probe-agent
echo "   Binary: $AGENT_DIR/probe-agent"

echo "[4/4] Writing config..."
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

echo "Starting probe-agent..."
echo "==================================="
export AGENT_CONFIG_PATH="$AGENT_DIR/config.yaml"
exec "$AGENT_DIR/probe-agent" run
