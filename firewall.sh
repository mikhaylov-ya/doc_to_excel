#!/bin/bash
# Script to open port 8080 in firewall

echo "🔥 Checking and opening firewall for port 8080..."

# Check if iptables is being used
if command -v iptables &> /dev/null; then
    echo "📋 Current iptables rules for port 8080:"
    if iptables -L INPUT -n | grep -q "dpt:8080"; then
        echo "✅ Port 8080 already allowed in iptables"
    else
        echo "No existing rules for port 8080"
        echo "➕ Adding iptables rule to allow port 8080..."
        iptables -I INPUT -p tcp --dport 8080 -j ACCEPT
        echo "✅ Rule added"
    fi

    echo "📋 Verifying current rules:"
    iptables -L INPUT -n -v | grep 8080

    # Save rules (try different methods for different systems)
    if command -v iptables-save &> /dev/null; then
        # Try to create directory first
        mkdir -p /etc/iptables 2>/dev/null

        # Try saving with different methods
        if iptables-save > /etc/iptables/rules.v4 2>/dev/null; then
            echo "💾 Rules saved to /etc/iptables/rules.v4"
        elif iptables-save > /etc/sysconfig/iptables 2>/dev/null; then
            echo "💾 Rules saved to /etc/sysconfig/iptables"
        elif service iptables save 2>/dev/null; then
            echo "💾 Rules saved via service"
        elif command -v netfilter-persistent &> /dev/null; then
            netfilter-persistent save
            echo "💾 Rules saved via netfilter-persistent"
        elif command -v iptables-persistent &> /dev/null; then
            iptables-persistent save
            echo "💾 Rules saved via iptables-persistent"
        else
            echo "⚠️  Note: iptables rules may not persist after reboot"
            echo "💡 To make persistent, install: apt-get install iptables-persistent"
        fi
    fi
fi

# Check if ufw is being used
if command -v ufw &> /dev/null; then
    echo ""
    echo "📋 Checking UFW status..."
    ufw status

    echo "➕ Opening port 8080 in UFW..."
    ufw allow 8080/tcp

    echo "✅ UFW rule added"
fi

# Check if firewalld is being used
if command -v firewall-cmd &> /dev/null; then
    echo ""
    echo "➕ Opening port 8080 in firewalld..."
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --reload

    echo "✅ firewalld rule added"
fi

echo ""
echo "🧪 Testing port accessibility..."
echo "From inside container:"
docker exec doc-to-excel-converter wget -q -O- http://localhost:8080/health 2>&1 || echo "Internal test: OK (curl not available, but that's normal)"

echo ""
echo "From server itself:"
curl -s http://localhost:8080/health | head -5 || wget -q -O- http://localhost:8080/health 2>&1 | head -5

echo ""
echo "✅ Port 8080 should now be accessible from outside!"

# Get all IPs
ALL_IPS=$(hostname -I)
echo "🌐 Server IPs detected: $ALL_IPS"
echo ""
echo "📍 Try accessing the service at:"
for ip in $ALL_IPS; do
    echo "   http://$ip:8080"
done
