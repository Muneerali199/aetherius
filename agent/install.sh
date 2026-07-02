#!/usr/bin/env bash
set -euo pipefail

# Aetherius Node Agent Installer
# Detects OS, installs dependencies, and registers the node

AETHERIUS_SERVER="${AETHERIUS_SERVER:-https://api.aetherius.io}"
AETHERIUS_PROVIDER_ID="${AETHERIUS_PROVIDER_ID:-}"
AETHERIUS_AGENT_VERSION="0.1.0"
INSTALL_DIR="/opt/aetherius"

echo "=== Aetherius Node Agent Installer v${AETHERIUS_AGENT_VERSION} ==="
echo ""

# Root check
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo)"
  exit 1
fi

# Provider ID check
if [ -z "$AETHERIUS_PROVIDER_ID" ]; then
  echo "Error: AETHERIUS_PROVIDER_ID is not set"
  echo "Get your provider ID from https://app.aetherius.io/settings/nodes"
  exit 1
fi

echo "Installing system dependencies..."

# OS detection
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)
    if command -v apt-get &>/dev/null; then
      apt-get update -qq
      apt-get install -y -qq curl ca-certificates
    elif command -v yum &>/dev/null; then
      yum install -y curl ca-certificates
    fi
    ;;
  Darwin)
    if ! command -v brew &>/dev/null; then
      echo "Installing Homebrew..."
      /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Check for NVIDIA drivers
if ! command -v nvidia-smi &>/dev/null; then
  echo "Warning: NVIDIA drivers not found"
  echo "Install NVIDIA drivers from: https://www.nvidia.com/drivers"
fi

# Check for Docker
if ! command -v docker &>/dev/null; then
  echo "Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable --now docker
fi

# Check for NVIDIA Container Toolkit
if ! command -v nvidia-ctk &>/dev/null; then
  echo "Installing NVIDIA Container Toolkit..."
  if command -v apt-get &>/dev/null; then
    distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
    curl -sL "https://github.com/NVIDIA/nvidia-container-runtime/raw/main/gpg" | gpg --dearmor -o /usr/share/keyrings/nvidia-docker.gpg
    apt-get update -qq
    apt-get install -y -qq nvidia-container-toolkit
    nvidia-ctk runtime configure --runtime=docker
    systemctl restart docker
  fi
fi

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download agent binary
echo "Downloading agent binary..."
BINARY_URL="${AETHERIUS_SERVER}/downloads/agent/latest/aetherius-agent-${OS,,}-${ARCH}"
curl -fsSL "$BINARY_URL" -o "$INSTALL_DIR/aetherius-agent"
chmod +x "$INSTALL_DIR/aetherius-agent"

# Create systemd service (Linux)
if [ "$OS" = "Linux" ]; then
  cat > /etc/systemd/system/aetherius-agent.service <<EOF
[Unit]
Description=Aetherius Node Agent
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=root
Environment=AETHERIUS_SERVER_URL=${AETHERIUS_SERVER}
Environment=AETHERIUS_PROVIDER_ID=${AETHERIUS_PROVIDER_ID}
Environment=AETHERIUS_AGENT_VERSION=${AETHERIUS_AGENT_VERSION}
ExecStart=${INSTALL_DIR}/aetherius-agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable --now aetherius-agent
  echo ""
  echo "Agent installed and running!"
  echo "Check status: sudo systemctl status aetherius-agent"
  echo "View logs: sudo journalctl -u aetherius-agent -f"
elif [ "$OS" = "Darwin" ]; then
  # Create launchd plist for macOS
  mkdir -p ~/Library/LaunchAgents
  cat > ~/Library/LaunchAgents/io.aetherius.agent.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.aetherius.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/aetherius-agent</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>AETHERIUS_SERVER_URL</key>
        <string>${AETHERIUS_SERVER}</string>
        <key>AETHERIUS_PROVIDER_ID</key>
        <string>${AETHERIUS_PROVIDER_ID}</string>
        <key>AETHERIUS_AGENT_VERSION</key>
        <string>${AETHERIUS_AGENT_VERSION}</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${INSTALL_DIR}/agent.log</string>
    <key>StandardErrorPath</key>
    <string>${INSTALL_DIR}/agent.log</string>
</dict>
</plist>
EOF

  launchctl load ~/Library/LaunchAgents/io.aetherius.agent.plist
  echo ""
  echo "Agent installed and running!"
  echo "View logs: tail -f ${INSTALL_DIR}/agent.log"
fi

echo ""
echo "=== Installation Complete ==="
echo "Node will appear in your Aetherius dashboard shortly."
