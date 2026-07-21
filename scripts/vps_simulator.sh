#!/bin/bash
echo "🚀 Preparando Máquina Virtual (VPS) de pruebas en Docker..."

# 1. Crear directorio y llaves si no existen
mkdir -p .mock_ssh
if [ ! -f ".mock_ssh/id_rsa" ]; then
    ssh-keygen -t rsa -b 4096 -f .mock_ssh/id_rsa -N "" -q
    echo "🔑 Nuevas llaves SSH generadas en .mock_ssh/"
fi
PUB_KEY=$(cat .mock_ssh/id_rsa.pub)

# 2. Limpiar contenedor previo
docker rm -f tarhiata_vps_sim 2>/dev/null || true

# 3. Lanzar Contenedor (Ubuntu + SSH + Docker Sock)
echo "📦 Iniciando contenedor Ubuntu con OpenSSH..."
docker run -d --name tarhiata_vps_sim --privileged \
  -p 2222:22 -p 8080:80 -p 8443:443 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ubuntu:22.04 \
  bash -c "apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y openssh-server sudo curl docker.io iptables ufw && mkdir -p /run/sshd && mkdir -p /root/.ssh && echo \"$PUB_KEY\" > /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys && /usr/sbin/sshd -D"

echo ""
echo "✅ Simulador Listo y corriendo de fondo!"
echo "Para conectarte desde Tarhiata-ops, ve a 'Configurar Credenciales' e ingresa:"
echo "---------------------------------------------------"
echo "👉 Host:         127.0.0.1"
echo "👉 Puerto:       2222"
echo "👉 Usuario:      root"
echo "👉 Ruta Llave:   $(pwd)/.mock_ssh/id_rsa"
echo "---------------------------------------------------"
echo "💡 Nota: Cuando corras el 'Bootstrapper', el paso de UFW (Firewall) podría soltar un warning porque estás en un contenedor, es normal. ¡Todo lo demás (Swarm, Traefik, Despliegues) funcionará de maravilla usando el motor Docker de tu Mac!"
