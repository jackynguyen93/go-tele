#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔═══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     Telegram Channel Monitor - Docker Setup          ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo -e "${RED}✗ config.yaml not found${NC}"
    echo ""
    echo "Creating config.yaml from example..."
    cp config.yaml.example config.yaml
    echo -e "${YELLOW}⚠ Please edit config.yaml with your Telegram credentials${NC}"
    echo ""
    echo "Required fields:"
    echo "  - telegram.api_id (from my.telegram.org)"
    echo "  - telegram.api_hash (from my.telegram.org)"
    echo "  - telegram.phone_number (or bot_token)"
    echo ""
    echo "Then run this script again."
    exit 1
fi

echo -e "${GREEN}✓ config.yaml found${NC}"

# Create data directory
mkdir -p data/tdlib data/files
echo -e "${GREEN}✓ Data directories created${NC}"

# Check if image exists
if docker images | grep -q "tdlib-go-tdclient"; then
    echo -e "${GREEN}✓ Docker image exists${NC}"

    echo ""
    read -p "Rebuild image? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Building Docker image...${NC}"
        docker compose build
    fi
else
    echo -e "${YELLOW}⚠ Docker image not found${NC}"
    echo -e "${BLUE}Building Docker image (this may take 5-10 minutes)...${NC}"
    docker compose build
fi

echo ""
echo -e "${GREEN}✓ Setup complete!${NC}"
echo ""
echo "Choose an option:"
echo "  1) Run in foreground (interactive)"
echo "  2) Run in background (detached)"
echo "  3) View logs of running container"
echo "  4) Stop container"
echo "  5) Exit"
echo ""
read -p "Enter choice [1-5]: " choice

case $choice in
    1)
        echo -e "${BLUE}Starting in foreground...${NC}"
        echo "Press Ctrl+C to stop"
        echo ""
        docker compose up
        ;;
    2)
        echo -e "${BLUE}Starting in background...${NC}"
        docker compose up -d
        echo ""
        echo -e "${GREEN}✓ Container started${NC}"
        echo ""
        echo "Useful commands:"
        echo "  View logs:      docker compose logs -f"
        echo "  Attach to CLI:  docker attach telegram-channel-monitor"
        echo "  Stop:           docker compose stop"
        ;;
    3)
        echo -e "${BLUE}Showing logs (Ctrl+C to exit)...${NC}"
        docker compose logs -f
        ;;
    4)
        echo -e "${BLUE}Stopping container...${NC}"
        docker compose stop
        echo -e "${GREEN}✓ Container stopped${NC}"
        ;;
    5)
        echo "Goodbye!"
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac
