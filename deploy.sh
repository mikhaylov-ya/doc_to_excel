#!/bin/bash
# Deployment script for DOC to Excel Converter

set -e  # Exit on error

echo "🚀 Starting deployment..."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install Docker first.${NC}"
    echo "Run: curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed.${NC}"
    echo "Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

echo -e "${BLUE}📦 Building Docker image...${NC}"
docker-compose build

echo -e "${BLUE}🛑 Stopping existing containers...${NC}"
docker-compose down

echo -e "${BLUE}🚀 Starting new containers...${NC}"
docker-compose up -d

echo -e "${GREEN}✅ Deployment complete!${NC}"
echo ""
echo "📊 Container status:"
docker-compose ps

echo ""
echo "🌐 Service is now running at:"
echo "   http://localhost:8080"
echo "   http://$(hostname -I | awk '{print $1}'):8080"

echo ""
echo "📝 Useful commands:"
echo "   View logs:     docker-compose logs -f"
echo "   Stop service:  docker-compose stop"
echo "   Start service: docker-compose start"
echo "   Restart:       docker-compose restart"
echo "   Remove all:    docker-compose down"
