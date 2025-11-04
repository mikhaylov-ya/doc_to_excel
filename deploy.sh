#!/bin/bash
# Deployment script for DOC to Excel Converter

set -e  # Exit on error

echo "🚀 Starting deployment..."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
DEPLOY_PATH="${DEPLOY_PATH:-$(pwd)}"
GOMAXPROCS="${GOMAXPROCS:-1}"
BUILD_LOCALLY="${BUILD_LOCALLY:-false}"  # Set to 'true' to build on server, 'false' to use pre-built image

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

# Navigate to deployment directory
if [ "$DEPLOY_PATH" != "$(pwd)" ]; then
    echo -e "${BLUE}📁 Navigating to deployment directory: ${DEPLOY_PATH}${NC}"
    cd "$DEPLOY_PATH"
fi

# Pull latest changes if it's a git repository
if [ -d .git ]; then
    echo -e "${BLUE}🔄 Pulling latest changes from git...${NC}"
    git fetch origin
    git reset --hard origin/main
else
    echo -e "${YELLOW}⚠️  Not a git repository, skipping git pull${NC}"
fi

echo -e "${BLUE}🛑 Stopping existing containers...${NC}"
docker-compose down || true

# Check if we should build locally or use pre-built image
if [ "$BUILD_LOCALLY" = "true" ]; then
    echo -e "${BLUE}📦 Building Docker image locally with optimizations...${NC}"
    DOCKER_BUILDKIT=1 GOPROXY=https://proxy.golang.org,direct \
        docker-compose build --build-arg GOMAXPROCS=${GOMAXPROCS}
else
    echo -e "${YELLOW}📦 Skipping build - using pre-built image${NC}"
    echo -e "${YELLOW}💡 If image doesn't exist, set BUILD_LOCALLY=true${NC}"
fi

echo -e "${BLUE}🚀 Starting new containers...${NC}"
docker-compose up -d

echo -e "${BLUE}⏳ Waiting for service to be healthy...${NC}"
sleep 10

# Health check
echo -e "${BLUE}🏥 Checking health status...${NC}"
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Health check passed!${NC}"
else
    echo -e "${RED}⚠️  Health check failed, but service may still be starting...${NC}"
fi

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
