# 📄 DOC to Excel Converter

A web service that converts Word documents (.doc, .docx) to Excel spreadsheets (.xlsx) with parsed article data and references.

## 🌟 Features

- ✅ **Web Interface**: Simple drag-and-drop file upload
- ✅ **Auto-conversion**: Converts DOC/DOCX to structured Excel
- ✅ **REST API**: HTTP endpoint for programmatic access
- ✅ **Dockerized**: Easy deployment with Docker
- ✅ **Reference Parsing**: Extracts and parses bibliographic references
- ✅ **Multi-sheet Output**: Separate sheets for articles, references, and DOIs

## 🏗️ Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Browser   │ ◄─────► │  Go Server   │ ◄─────► │  Temp Files │
│  (Frontend) │  HTTP   │   (Gin)      │         │             │
└─────────────┘         └──────────────┘         └─────────────┘
```

## 📋 Requirements

### For Docker Deployment (Recommended)
- Docker Engine 20.10+
- Docker Compose 1.29+
- Linux server with 2GB+ RAM

### For Local Development
- Go 1.23+
- System dependencies: `poppler-utils`, `antiword`, `tesseract-ocr`

## 🚀 Quick Start (Docker - Recommended)

### 1. On Your Remote Server

```bash
# Install Docker (if not installed)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# Log out and back in

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 2. Upload Code to Server

```bash
# From your local machine
scp -r ./doc_to_excel user@your-server-ip:/home/user/

# OR use git
ssh user@your-server-ip
git clone <your-repo>
cd doc_to_excel
```

### 3. Deploy

```bash
# Run the deployment script
./deploy.sh

# OR manually:
docker-compose build
docker-compose up -d
```

### 4. Access the Service

```
http://your-server-ip:8080
```

## 📖 Usage

### Web Interface

1. Open `http://your-server-ip:8080` in your browser
2. Click or drag-and-drop a `.doc` or `.docx` file
3. Wait for conversion (progress indicator shown)
4. Download automatically starts when complete

### API Endpoint

**Upload and Convert:**

```bash
curl -X POST \
  http://your-server-ip:8080/api/convert \
  -F "document=@yourfile.docx" \
  --output result.xlsx
```

**Health Check:**

```bash
curl http://your-server-ip:8080/health
```

Response:
```json
{
  "status": "ok",
  "service": "doc-to-excel",
  "version": "1.0.0",
  "time": "2025-01-15T10:30:00Z"
}
```

## 🐳 Docker Commands

```bash
# View logs
docker-compose logs -f

# Stop service
docker-compose stop

# Start service
docker-compose start

# Restart service
docker-compose restart

# Check status
docker-compose ps

# Remove containers
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```

## 🔧 Configuration

### Environment Variables

Edit `docker-compose.yml`:

```yaml
environment:
  - PORT=8080           # Server port
  - GIN_MODE=release    # Gin mode (debug/release)
  - TZ=UTC              # Timezone
```

### Resource Limits

Adjust in `docker-compose.yml`:

```yaml
deploy:
  resources:
    limits:
      cpus: '1.5'      # Max CPU cores
      memory: 1G       # Max memory
```

### File Size Limit

Edit `server.go`:

```go
router.MaxMultipartMemory = 50 << 20  // 50 MB
```

## 📂 Project Structure

```
doc_to_excel/
├── server.go              # HTTP server (main entry point)
├── main.go                # Core processing logic
├── parse_reference.go     # Reference parsing
├── parse_web.go           # Web scraping
├── types.go               # Data types
├── frontend/
│   ├── index.html         # Web UI
│   └── static/            # Static assets
├── temp/                  # Temporary files
├── Dockerfile             # Docker build config
├── docker-compose.yml     # Docker Compose config
├── deploy.sh              # Deployment script
└── README.md              # This file
```

## 🔍 Monitoring

### View Logs

```bash
# All logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100

# Logs for specific time
docker-compose logs --since 10m
```

### Check Health

```bash
# Using curl
curl http://localhost:8080/health

# Using Docker
docker ps
docker inspect doc-to-excel-converter
```

## 🛠️ Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs

# Check port availability
sudo netstat -tlnp | grep 8080

# Rebuild from scratch
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Out of memory

Increase memory limit in `docker-compose.yml` or upgrade server RAM.

### Slow conversion

- Check server CPU/RAM usage
- Reduce concurrent requests
- Increase resource limits

### Permission errors

```bash
# Fix temp directory permissions
sudo chown -R $USER:$USER ./temp
chmod 755 ./temp
```

## 🔒 Security Notes

### For Production:

1. **Add HTTPS** (use nginx reverse proxy + Let's Encrypt)
2. **Add authentication** if needed
3. **Rate limiting** to prevent abuse
4. **Input validation** (already implemented for file types/sizes)
5. **Firewall rules** to restrict access

### Example nginx config (optional):

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # Increase timeout for large files
        proxy_read_timeout 300s;
        client_max_body_size 50M;
    }
}
```

## 📊 Performance

### Tested Specs:
- **Server**: 2 vCPU, 2GB RAM, 30GB SSD
- **Max concurrent users**: ~100
- **Average conversion time**: 5-15 seconds (depending on document size)
- **Max file size**: 50MB

### Optimization tips:
- Use SSD storage for faster temp file I/O
- Increase worker processes if needed
- Add caching for repeated conversions
- Use CDN for frontend assets (if needed)

## 🔄 Updates

To update the service:

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose up -d --build

# OR use deploy script
./deploy.sh
```

## 🐛 Known Issues

- Very large files (>50MB) may timeout
- Some complex DOC formats may have parsing issues
- Requires system fonts for proper character encoding

## 📝 License

MIT License - Feel free to modify and use

## 👨‍💻 Support

For issues or questions:
1. Check the logs: `docker-compose logs -f`
2. Verify health: `curl http://localhost:8080/health`
3. Review this README

## 🎯 Roadmap

- [ ] Add batch conversion (multiple files)
- [ ] Add progress percentage for large files
- [ ] Add conversion history
- [ ] Add user authentication (optional)
- [ ] Add email notifications for completed conversions
- [ ] Support more input formats (PDF, RTF)

---

**Built with ❤️ using Go, Gin, and Docker**
