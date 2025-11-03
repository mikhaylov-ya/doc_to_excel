# 🚀 Deployment Guide

Quick guide to deploy DOC to Excel Converter on your remote server.

## 📋 Prerequisites

- Remote server with:
  - **OS**: Ubuntu 20.04+ / Debian 10+ / CentOS 8+
  - **Specs**: 2 vCPU, 2GB RAM, 30GB SSD
  - **SSH access**: Configured
  - **Ports**: 8080 open (or any custom port)

## 🎯 Step-by-Step Deployment

### Step 1: Prepare Server

SSH into your server:

```bash
ssh user@your-server-ip
```

Update system packages:

```bash
sudo apt update && sudo apt upgrade -y
# OR for CentOS/RHEL
sudo yum update -y
```

### Step 2: Install Docker

```bash
# Download Docker installation script
curl -fsSL https://get.docker.com -o get-docker.sh

# Run installation
sudo sh get-docker.sh

# Add your user to docker group (avoid sudo)
sudo usermod -aG docker $USER

# Apply group changes (logout and login)
exit
# Then SSH back in
```

Verify Docker installation:

```bash
docker --version
# Should show: Docker version 20.10.x or higher
```

### Step 3: Install Docker Compose

```bash
# Download Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

# Make it executable
sudo chmod +x /usr/local/bin/docker-compose

# Verify
docker-compose --version
# Should show: docker-compose version 1.29.x or higher
```

### Step 4: Upload Your Code

**Option A: Using SCP (from your local machine)**

```bash
# Create archive
cd /Users/aroslavmihajlov/Documents
tar -czf doc_to_excel.tar.gz doc_to_excel/

# Upload to server
scp doc_to_excel.tar.gz user@your-server-ip:~/

# On server, extract
ssh user@your-server-ip
tar -xzf doc_to_excel.tar.gz
cd doc_to_excel
```

**Option B: Using Git**

```bash
# On server
git clone <your-repository-url>
cd doc_to_excel
```

**Option C: Manual file copy**

```bash
# From local machine
scp -r /Users/aroslavmihajlov/Documents/doc_to_excel user@your-server-ip:~/
```

### Step 5: Deploy

```bash
cd doc_to_excel

# Run deployment script
./deploy.sh

# OR manually:
docker-compose build
docker-compose up -d
```

Wait for build to complete (5-10 minutes first time).

### Step 6: Verify Deployment

Check if container is running:

```bash
docker-compose ps
```

Should show:
```
NAME                      STATUS    PORTS
doc-to-excel-converter   Up        0.0.0.0:8080->8080/tcp
```

Check logs:

```bash
docker-compose logs -f
```

Should see:
```
🚀 Server starting on http://0.0.0.0:8080
```

Test health endpoint:

```bash
curl http://localhost:8080/health
```

Should return:
```json
{
  "status": "ok",
  "service": "doc-to-excel",
  "version": "1.0.0"
}
```

### Step 7: Access from Browser

Open in your browser:

```
http://YOUR_SERVER_IP:8080
```

You should see the upload interface!

## 🔧 Configuration

### Change Port

Edit `docker-compose.yml`:

```yaml
ports:
  - "9000:8080"  # Change 9000 to your desired port
```

Then restart:

```bash
docker-compose down
docker-compose up -d
```

### Firewall Configuration

**Ubuntu/Debian (UFW):**

```bash
sudo ufw allow 8080/tcp
sudo ufw enable
sudo ufw status
```

**CentOS/RHEL (firewalld):**

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

**Cloud Providers:**
- **AWS**: Add inbound rule in Security Group
- **DigitalOcean**: Add firewall rule in Networking
- **Hetzner**: Configure firewall in Cloud Console

## 📊 Management

Use the management script:

```bash
./manage.sh
```

Interactive menu:
```
1) Start service
2) Stop service
3) Restart service
4) View logs
5) Check status
6) Update and rebuild
7) Clean up
8) Exit
```

## 🔄 Updates

When you have new code:

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# OR use management script
./manage.sh
# Select option 6 (Update and rebuild)
```

## 🛡️ Security Best Practices

### 1. Use Firewall

Only allow necessary ports:

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 8080/tcp
sudo ufw enable
```

### 2. Keep System Updated

```bash
# Set up automatic updates (Ubuntu)
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

### 3. Monitor Logs

```bash
# Watch for suspicious activity
docker-compose logs -f | grep -i error
```

### 4. Backup Configuration

```bash
# Backup important files
tar -czf backup-$(date +%F).tar.gz \
  docker-compose.yml \
  server.go \
  frontend/
```

## 🐛 Troubleshooting

### Problem: Port already in use

```bash
# Check what's using port 8080
sudo lsof -i :8080
# OR
sudo netstat -tlnp | grep 8080

# Kill the process or change port in docker-compose.yml
```

### Problem: Container keeps restarting

```bash
# Check logs
docker-compose logs

# Common issues:
# - Out of memory: Increase server RAM or reduce limits
# - Permission error: Fix temp/ directory permissions
# - Port conflict: Change port mapping
```

### Problem: Cannot access from browser

1. Check firewall rules
2. Verify container is running: `docker-compose ps`
3. Check logs: `docker-compose logs`
4. Try from server itself: `curl http://localhost:8080`

### Problem: Slow conversion

```bash
# Check resource usage
docker stats

# Increase limits in docker-compose.yml:
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
```

## 📈 Monitoring

### Check Service Health

```bash
# Health check
curl http://localhost:8080/health

# Container stats
docker stats doc-to-excel-converter

# Disk usage
docker system df
```

### View Logs

```bash
# Last 100 lines
docker-compose logs --tail=100

# Follow logs
docker-compose logs -f

# Filter by time
docker-compose logs --since 1h
```

## 🔄 Backup & Restore

### Backup

```bash
# Backup uploaded files (if needed)
tar -czf backup-temp-$(date +%F).tar.gz temp/

# Backup entire application
cd ..
tar -czf doc-to-excel-backup-$(date +%F).tar.gz doc_to_excel/
```

### Restore

```bash
# Extract backup
tar -xzf doc-to-excel-backup-YYYY-MM-DD.tar.gz

# Rebuild containers
cd doc_to_excel
docker-compose up -d --build
```

## 📞 Support Checklist

If something goes wrong, gather this info:

```bash
# 1. Docker version
docker --version
docker-compose --version

# 2. Container status
docker-compose ps

# 3. Logs
docker-compose logs --tail=200

# 4. Resource usage
docker stats --no-stream

# 5. System info
free -h
df -h
top -bn1 | head -20
```

## ✅ Post-Deployment Checklist

- [ ] Container is running (`docker-compose ps`)
- [ ] Health check passes (`curl http://localhost:8080/health`)
- [ ] Can access web interface from browser
- [ ] Can upload and convert a test file
- [ ] Firewall configured correctly
- [ ] Auto-restart configured (`restart: unless-stopped`)
- [ ] Logs are accessible
- [ ] Monitoring set up (optional)

---

**Deployment complete! 🎉**

Your DOC to Excel Converter is now running at: `http://YOUR_SERVER_IP:8080`
