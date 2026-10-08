#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "  DioramaOps VPS Automated Deployment"
echo "=========================================="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 1. Pastikan file .env ada
if [ ! -f .env ]; then
  echo "File .env tidak ditemukan. Membuat dari .env.example..."
  cp .env.example .env
  
  # Generate random ADMIN_SECRET jika default
  RANDOM_SECRET=$(openssl rand -hex 16 2>/dev/null || date +%s%N | sha256sum | head -c 32)
  sed -i "s/ADMIN_SECRET=change_this_to_a_secure_random_secret_token/ADMIN_SECRET=$RANDOM_SECRET/" .env
  echo "Generated random ADMIN_SECRET di .env: $RANDOM_SECRET"
fi

# 2. Siapkan folder data dan logos untuk user distroless non-root (UID 65532)
echo "Menyiapkan volume direktori..."
mkdir -p data logos 2>/dev/null || true
chmod 777 data logos 2>/dev/null || true

# 3. Build container secara bertahap (sekuensial di dalam Dockerfile) untuk mencegah freeze VPS
echo ""
echo "Membangun container image (sekuensial untuk menghemat RAM/CPU VPS)..."
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1
docker compose build dioramaops

echo ""
echo "Menjalankan container..."
docker compose up -d --remove-orphans

# 4. Verifikasi status container
echo ""
echo "Menunggu container siap..."
sleep 3
docker compose ps

echo "=========================================="
echo "DioramaOps berhasil di-deploy!"
echo "Port: 127.0.0.1:7437"
echo "Reverse Proxy Host Caddy siap meneruskan traffic ke 127.0.0.1:7437"
echo "=========================================="
