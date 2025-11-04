#!/bin/bash
# Script to set up domain with Nginx reverse proxy and SSL

set -e

# Load configuration from config.env
if [ -f "config.env" ]; then
    echo "📋 Loading config from config.env..."
    source config.env
fi

# Validate configuration
if [ -z "$DOMAIN" ]; then
    echo "❌ Error: DOMAIN not set in config.env"
    echo "Create config.env with: DOMAIN=yourdomain.com"
    exit 1
fi

if [ -z "$EMAIL" ]; then
    EMAIL="admin@$DOMAIN"
fi

echo "🌐 Setting up domain: $DOMAIN"
echo "📧 Using email: $EMAIL"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root or with sudo"
    exit 1
fi

# Install Nginx if not present
if ! command -v nginx &> /dev/null; then
    echo "📦 Installing Nginx..."
    apt-get update
    apt-get install -y nginx
else
    echo "✅ Nginx already installed"
fi

# Install Certbot for SSL
if ! command -v certbot &> /dev/null; then
    echo "🔒 Installing Certbot for SSL..."
    apt-get install -y certbot python3-certbot-nginx
else
    echo "✅ Certbot already installed"
fi

# Create Nginx configuration from template
echo "⚙️  Creating Nginx configuration..."

if [ ! -f "nginx.conf.template" ]; then
    echo "❌ Error: nginx.conf.template not found"
    echo "💡 Make sure you're running this script from the deployment directory"
    exit 1
fi

# Copy template and replace placeholders
sed "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" nginx.conf.template > /etc/nginx/sites-available/$DOMAIN

echo "✅ Created /etc/nginx/sites-available/$DOMAIN"

# Enable site
ln -sf /etc/nginx/sites-available/$DOMAIN /etc/nginx/sites-enabled/

# Remove default site if it exists
rm -f /etc/nginx/sites-enabled/default

# Test Nginx configuration
echo "🧪 Testing Nginx configuration..."
nginx -t

# Restart Nginx
echo "🔄 Restarting Nginx..."
systemctl restart nginx
systemctl enable nginx

echo ""
echo "✅ Nginx configured successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Make sure your DNS A record points to this server's IP"
echo "2. Wait a few minutes for DNS propagation"
echo "3. Test HTTP access: http://$DOMAIN"
echo ""
echo "🔒 To set up HTTPS/SSL, run:"
echo "   sudo certbot --nginx -d $DOMAIN --non-interactive --agree-tos -m $EMAIL"
echo ""
echo "Or run this command now to set up SSL automatically:"
read -p "Set up SSL now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🔒 Setting up SSL certificate..."
    certbot --nginx -d $DOMAIN --non-interactive --agree-tos -m $EMAIL

    echo ""
    echo "✅ SSL certificate installed!"
    echo "🌐 Your service is now available at: https://$DOMAIN"
else
    echo "⏭️  Skipping SSL setup. You can run it later with:"
    echo "   sudo certbot --nginx -d $DOMAIN"
fi

echo ""
echo "🎉 Setup complete!"
echo "📍 Service URL: http://$DOMAIN (or https://$DOMAIN if SSL is configured)"
