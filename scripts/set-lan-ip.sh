#!/usr/bin/env bash
# Detect local IP and update .env for mobile/LAN access.
# Run from project root: ./scripts/set-lan-ip.sh

set -e
cd "$(dirname "$0")/.."

# Detect local IP (works on macOS and Linux)
detect_ip() {
  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS: try en0 (Wi-Fi) first, then other interfaces
    ip=$(ipconfig getifaddr en0 2>/dev/null) || \
    ip=$(ipconfig getifaddr en1 2>/dev/null) || \
    ip=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1)
  else
    # Linux
    ip=$(hostname -I 2>/dev/null | awk '{print $1}') || \
    ip=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
  fi
  echo "${ip:-127.0.0.1}"
}

IP=$(detect_ip)
echo "Detected IP: $IP"

ENV_FILE=".env"
if [[ ! -f "$ENV_FILE" ]]; then
  cp .env.example "$ENV_FILE" 2>/dev/null || true
fi

# Update or add LAN variables
for var in MINIO_PUBLIC_BASE_URL MINIO_PUBLIC_URL VITE_API_URL CORS_ORIGIN; do
  case "$var" in
    MINIO_PUBLIC_BASE_URL|MINIO_PUBLIC_URL)
      value="http://${IP}:9000"
      ;;
    VITE_API_URL)
      value="http://${IP}:8080"
      ;;
    CORS_ORIGIN)
      value="http://${IP}:3000"
      ;;
    *)
      continue
      ;;
  esac
  if grep -q "^${var}=" "$ENV_FILE" 2>/dev/null; then
    if [[ "$(uname)" == "Darwin" ]]; then
      sed -i '' "s|^${var}=.*|${var}=${value}|" "$ENV_FILE"
    else
      sed -i "s|^${var}=.*|${var}=${value}|" "$ENV_FILE"
    fi
  else
    echo "${var}=${value}" >> "$ENV_FILE"
  fi
done

echo "Updated .env with IP $IP"
echo "  VITE_API_URL=$IP:8080"
echo "  CORS_ORIGIN=$IP:3000"
echo "  MINIO_PUBLIC_*=$IP:9000"
echo ""
echo "Run 'make rebuild' to apply. Then open http://${IP}:3000 on your mobile."
