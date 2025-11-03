#!/bin/bash
# Management script for DOC to Excel Converter

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

function show_menu() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}  DOC to Excel Converter${NC}"
    echo -e "${BLUE}================================${NC}"
    echo ""
    echo "1) Start service"
    echo "2) Stop service"
    echo "3) Restart service"
    echo "4) View logs"
    echo "5) Check status"
    echo "6) Update and rebuild"
    echo "7) Clean up (remove containers)"
    echo "8) Exit"
    echo ""
    echo -n "Select option: "
}

function start_service() {
    echo -e "${BLUE}🚀 Starting service...${NC}"
    docker-compose up -d
    echo -e "${GREEN}✅ Service started${NC}"
    show_status
}

function stop_service() {
    echo -e "${YELLOW}🛑 Stopping service...${NC}"
    docker-compose stop
    echo -e "${GREEN}✅ Service stopped${NC}"
}

function restart_service() {
    echo -e "${YELLOW}🔄 Restarting service...${NC}"
    docker-compose restart
    echo -e "${GREEN}✅ Service restarted${NC}"
    show_status
}

function view_logs() {
    echo -e "${BLUE}📋 Showing logs (Ctrl+C to exit)...${NC}"
    docker-compose logs -f --tail=100
}

function show_status() {
    echo -e "${BLUE}📊 Service status:${NC}"
    docker-compose ps
    echo ""

    if docker-compose ps | grep -q "Up"; then
        IP=$(hostname -I | awk '{print $1}')
        echo -e "${GREEN}🌐 Service is running at:${NC}"
        echo "   http://localhost:8080"
        echo "   http://${IP}:8080"
    else
        echo -e "${RED}⚠️  Service is not running${NC}"
    fi
}

function update_rebuild() {
    echo -e "${BLUE}📦 Updating and rebuilding...${NC}"
    git pull
    docker-compose down
    docker-compose build --no-cache
    docker-compose up -d
    echo -e "${GREEN}✅ Update complete${NC}"
    show_status
}

function cleanup() {
    echo -e "${YELLOW}🗑️  Cleaning up...${NC}"
    docker-compose down
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Main loop
while true; do
    show_menu
    read -r choice

    case $choice in
        1) start_service ;;
        2) stop_service ;;
        3) restart_service ;;
        4) view_logs ;;
        5) show_status ;;
        6) update_rebuild ;;
        7) cleanup ;;
        8) echo -e "${GREEN}Goodbye!${NC}"; exit 0 ;;
        *) echo -e "${RED}Invalid option${NC}" ;;
    esac

    echo ""
    echo -n "Press Enter to continue..."
    read -r
    clear
done
