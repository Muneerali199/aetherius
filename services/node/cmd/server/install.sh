#!/bin/bash
set -e

AETHERIUS_API="${AETHERIUS_API:-http://localhost:8082}"
AETHERIUS_TOKEN="${AETHERIUS_TOKEN:-}"
NODE_NAME="${NODE_NAME:-$(hostname)-$(date +%s)}"

if [ -z "$AETHERIUS_TOKEN" ]; then
  echo "ERROR: AETHERIUS_TOKEN is required"
  echo "Usage: AETHERIUS_TOKEN=your_api_key curl -fsSL http://localhost:8082/install.sh | sh"
  echo "Get your API key from: http://localhost:3000/billing"
  exit 1
fi

echo "=== Aetherius Node Agent Installer ==="
echo "Registering node: $NODE_NAME"
echo "API endpoint: $AETHERIUS_API"

detect_hardware() {
  OS_NAME=$(uname -s)
  ARCH=$(uname -m)
  CPU_MODEL=""
  CPU_CORES=${CPU_CORES:-0}
  TOTAL_RAM_GB=${TOTAL_RAM_GB:-0}
  TOTAL_DISK_GB=${TOTAL_DISK_GB:-0}
  GPU_MODELS="[]"
  TOTAL_GPU=${TOTAL_GPU:-0}
  TOTAL_VRAM_GB=${TOTAL_VRAM_GB:-0}

  case "$OS_NAME" in
    Linux)
      CPU_MODEL=$(grep -m1 "model name" /proc/cpuinfo | cut -d: -f2 | xargs 2>/dev/null || echo "Unknown CPU")
      CPU_CORES=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 1)
      TOTAL_RAM_GB=$(free -g | awk '/^Mem:/{print $2}' 2>/dev/null || echo 1)
      TOTAL_DISK_GB=$(df -BG / | awk 'NR==2{print $2}' | sed 's/G//' 2>/dev/null || echo 10)
      if command -v nvidia-smi &>/dev/null; then
        TOTAL_GPU=$(nvidia-smi -L | wc -l 2>/dev/null || echo 0)
        GPU_MODELS=$(nvidia-smi -L 2>/dev/null | awk -F': ' '{print $2}' | awk '{print $1}' | jq -R -s -c 'split("\n")[:-1]' 2>/dev/null || echo "[]")
        TOTAL_VRAM_GB=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | awk '{s+=$1} END{printf "%.0f", s/1024}' || echo 0)
      fi
      ;;
    Darwin)
      CPU_MODEL=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Apple Silicon")
      CPU_CORES=$(sysctl -n hw.ncpu 2>/dev/null || echo 1)
      TOTAL_RAM_GB=$(sysctl -n hw.memsize 2>/dev/null | awk '{print int($1/1073741824)}' || echo 1)
      TOTAL_DISK_GB=$(df -g / 2>/dev/null | awk 'NR==2{print $2; exit}' || echo 10)
      [ -z "$TOTAL_DISK_GB" ] && TOTAL_DISK_GB=10
      if command -v system_profiler &>/dev/null; then
        GPU_COUNT=$(system_profiler SPDisplaysDataType 2>/dev/null | grep -c "Chipset Model" || echo 0)
        if [ "$GPU_COUNT" -gt 0 ]; then
          TOTAL_GPU=$GPU_COUNT
          GPU_MODELS=$(system_profiler SPDisplaysDataType 2>/dev/null | grep "Chipset Model" | awk -F': ' '{print $2}' | jq -R -s -c 'split("\n")[:-1]' 2>/dev/null || echo "[]")
          VRAM_MB=$(system_profiler SPDisplaysDataType 2>/dev/null | grep "VRAM" | grep -oE '[0-9]+' | head -1 || echo 0)
          TOTAL_VRAM_GB=$(( (VRAM_MB + 1023) / 1024 ))
        fi
      fi
      ;;
  esac

  [ "$CPU_CORES" -lt 1 ] && CPU_CORES=1
  [ "$TOTAL_RAM_GB" -lt 1 ] && TOTAL_RAM_GB=1
  [ "$TOTAL_DISK_GB" -lt 1 ] && TOTAL_DISK_GB=10
  return 0
}

register_node() {
  echo "Detecting hardware..."
  detect_hardware

  echo "CPU: $CPU_MODEL ($CPU_CORES cores)"
  echo "RAM: ${TOTAL_RAM_GB}GB"
  echo "Disk: ${TOTAL_DISK_GB}GB"
  echo "GPU: $TOTAL_GPU ($TOTAL_VRAM_GB GB VRAM)"

  # Convert GB to bytes
  RAM_BYTES=$((TOTAL_RAM_GB * 1024 * 1024 * 1024))
  DISK_BYTES=$((TOTAL_DISK_GB * 1024 * 1024 * 1024))
  VRAM_BYTES=$((TOTAL_VRAM_GB * 1024 * 1024 * 1024))

  PAYLOAD=$(cat <<EOF
{
  "provider_id": "$(uuidgen 2>/dev/null || echo "00000000-0000-0000-0000-000000000001")",
  "agent_version": "1.0.0",
  "public_ip": "auto",
  "hardware": {
    "gpus": $(if [ "$TOTAL_GPU" -gt 0 ]; then
      echo "["
      FIRST=true
      for i in $(seq 1 $TOTAL_GPU); do
        $FIRST || echo ","
        FIRST=false
        GPU_VRAM=$((VRAM_BYTES / TOTAL_GPU))
        echo "{\"model\":\"$(echo "$GPU_MODELS" | jq -r ".[$((i-1))]" 2>/dev/null || echo "Unknown GPU")\",\"vram_bytes\":$GPU_VRAM,\"cores\":16384}"
      done
      echo "]"
    else
      echo "[]"
    fi),
    "cpu": {
      "model": "$CPU_MODEL",
      "cores": $CPU_CORES
    },
    "ram": {
      "total_bytes": $RAM_BYTES
    },
    "disk": {
      "total_bytes": $DISK_BYTES,
      "filesystem": "ext4"
    },
    "network": {
      "public_ip": "auto",
      "provider_ip": "auto"
    },
    "cuda_version": "12.5",
    "docker_version": "26.0",
    "os_name": "$OS_NAME"
  }
}
EOF
)

  echo "Registering node..."
  RESPONSE=$(curl -s -X POST "$AETHERIUS_API/v1/nodes/register" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $AETHERIUS_TOKEN" \
    -d "$PAYLOAD")

  NODE_ID=$(echo "$RESPONSE" | grep -o '"node_id":"[^"]*"' | head -1 | cut -d'"' -f4)

  if [ -z "$NODE_ID" ]; then
    echo "ERROR: Registration failed. Response:"
    echo "$RESPONSE"
    exit 1
  fi

  echo "Node registered successfully!"
  echo "Node ID: $NODE_ID"
  echo "$NODE_ID" > /tmp/aetherius_node_id
}

start_heartbeat() {
  NODE_ID=$(cat /tmp/aetherius_node_id 2>/dev/null)
  if [ -z "$NODE_ID" ]; then
    echo "ERROR: No node ID found. Run registration first."
    exit 1
  fi

  echo "Starting heartbeat loop (every 30s)..."
  echo "Press Ctrl+C to stop."

  while true; do
    RESPONSE=$(curl -s -X POST "$AETHERIUS_API/v1/nodes/$NODE_ID/heartbeat" \
      -H "Authorization: Bearer $AETHERIUS_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"status":"active"}')

    if echo "$RESPONSE" | grep -q '"status":"ok"'; then
      echo "[$(date +%H:%M:%S)] Heartbeat OK"
    else
      echo "[$(date +%H:%M:%S)] Heartbeat failed: $RESPONSE"
    fi

    sleep 30
  done
}

register_node
start_heartbeat
