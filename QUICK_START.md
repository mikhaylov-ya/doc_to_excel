# ⚡ Quick Start Guide

Get your DOC to Excel Converter running in 5 minutes!

## 🚀 Fastest Path to Deployment

### On Your Remote Server (2 vCPU, 2GB RAM)

```bash
# 1. Install Docker (one command)
curl -fsSL https://get.docker.com | sudo sh

# 2. Add Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 3. Upload your code (from local machine)
scp -r /Users/aroslavmihajlov/Documents/doc_to_excel user@YOUR_SERVER_IP:~/

# 4. Deploy (on server)
cd doc_to_excel
./deploy.sh

# 5. Open firewall (Ubuntu)
sudo ufw allow 8080/tcp
```

**Done!** Access at: `http://YOUR_SERVER_IP:8080`

## 📱 Test It

```bash
# Health check
curl http://YOUR_SERVER_IP:8080/health

# Upload test file
curl -X POST \
  http://YOUR_SERVER_IP:8080/api/convert \
  -F "document=@test.docx" \
  --output result.xlsx
```

## 🎯 What You Get

- ✅ **Web UI**: Drag & drop interface at `http://YOUR_SERVER_IP:8080`
- ✅ **REST API**: `/api/convert` endpoint
- ✅ **Auto-restart**: Container restarts if crashed
- ✅ **Resource limits**: Protected from memory/CPU abuse
- ✅ **Health monitoring**: `/health` endpoint

## 📊 Manage Your Service

```bash
cd doc_to_excel

# View logs
docker-compose logs -f

# Restart
docker-compose restart

# Stop
docker-compose stop

# Start
docker-compose start

# OR use interactive menu
./manage.sh
```

## 🔧 Common Tasks

### Change Port (from 8080 to 9000)

Edit `docker-compose.yml`:
```yaml
ports:
  - "9000:8080"
```

Then:
```bash
docker-compose down
docker-compose up -d
```

### Update Code

```bash
git pull
docker-compose up -d --build
```

### View Server IP

```bash
hostname -I
```

## 🐛 Quick Troubleshooting

**Container not starting?**
```bash
docker-compose logs
```

**Port already in use?**
```bash
sudo lsof -i :8080
# Change port in docker-compose.yml
```

**Cannot access from browser?**
```bash
# Check firewall
sudo ufw status

# Allow port
sudo ufw allow 8080/tcp
```

**Slow performance?**
```bash
# Check resources
docker stats
```

## 📚 Next Steps

1. ✅ Service is running
2. 📖 Read full [README.md](README.md) for details
3. 🚀 Check [DEPLOYMENT.md](DEPLOYMENT.md) for advanced config
4. 🔒 Consider adding HTTPS (nginx + Let's Encrypt)

## 💡 Pro Tips

- Use `./manage.sh` for easy management
- Monitor with `docker stats`
- Backup with `docker-compose down` before updates
- Keep system updated: `sudo apt update && sudo apt upgrade`

---

**That's it! Your service is live! 🎉**

Need help? Check logs: `docker-compose logs -f`
