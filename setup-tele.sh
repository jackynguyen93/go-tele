#!/bin/bash

# Telegram Channel Monitor - Server Setup Script
# Registry: ghcr.io/jackynguyen93

set -e  # Exit on error

# Configuration - CHANGE THESE VALUES
REGISTRY_IMAGE="ghcr.io/jackynguyen93/tdclient:latest"
API_ID="YOUR_API_ID"              # Get from https://my.telegram.org
API_HASH="YOUR_API_HASH"          # Get from https://my.telegram.org

# Optional: GitHub token for private images
# GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# Project directory
PROJECT_DIR="$HOME/telegram-monitor"

echo "========================================="
echo "Telegram Channel Monitor Setup"
echo "========================================="
echo "Registry: ${REGISTRY_IMAGE}"
echo "Installing to: ${PROJECT_DIR}"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed"
    echo "Please install Docker first: https://docs.docker.com/engine/install/"
    exit 1
fi

# Check if docker compose is available
if ! docker compose version &> /dev/null; then
    echo "Error: docker compose is not available"
    echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi

# Create project directory
echo "Creating project directory..."
mkdir -p "${PROJECT_DIR}"
cd "${PROJECT_DIR}"

# Login to GitHub Container Registry (if GITHUB_TOKEN is set)
if [ ! -z "$GITHUB_TOKEN" ]; then
    echo "Logging in to GitHub Container Registry..."
    echo "$GITHUB_TOKEN" | docker login ghcr.io -u jackynguyen93 --password-stdin
fi

# Create config.yaml
echo "Creating config.yaml..."
cat > config.yaml << EOF
telegram:
  api_id: ${API_ID}
  api_hash: "${API_HASH}"
  phone_number: ""
  use_test_dc: false

database:
  type: "sqlite"
  dsn: "data/messages.db"

tdlib:
  database_directory: "data/tdlib"
  files_directory: "data/files"
  use_file_database: true
  use_chat_info_database: true
  use_message_database: true
  system_language: "en"
  device_model: "Server"
  system_version: "Linux"
  app_version: "1.0.0"

channels:
  - "telegram"
  - "durov"

logging:
  level: "info"
  format: "text"
EOF

# Create docker-compose.yml
echo "Creating docker-compose.yml..."
cat > docker-compose.yml << EOF
services:
  tdclient:
    image: ${REGISTRY_IMAGE}
    container_name: telegram-channel-monitor
    restart: unless-stopped

    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data

    stdin_open: true
    tty: true

    environment:
      - LOG_LEVEL=info

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF

# Create data directories
echo "Creating data directories..."
mkdir -p data/tdlib data/files
chmod -R 755 data

# Pull the image
echo "Pulling Docker image..."
docker pull "${REGISTRY_IMAGE}"

# Set proper permissions
chmod 600 config.yaml

echo ""
echo "========================================="
echo "Setup Complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Edit config.yaml and add your API credentials:"
echo "   cd ${PROJECT_DIR}"
echo "   nano config.yaml"
echo ""
echo "2. Run for first-time authentication (QR code):"
echo "   docker compose run --rm tdclient"
echo ""
echo "3. Scan the QR code with your Telegram app"
echo ""
echo "4. After authentication, run in background:"
echo "   docker compose up -d"
echo ""
echo "5. Check logs:"
echo "   docker compose logs -f"
echo ""
echo "========================================="
