#!/bin/bash
# Запускается на сервере. Pull latest, build, restart systemd unit.
# Миграции запускаются ВРУЧНУЮ перед этим скриптом, если в push'е есть новая 000NNN_*.up.sql:
#
#   cd /root/lana-flowers-go && git fetch && git reset --hard origin/main \
#     && go build -o app ./cmd/api \
#     && ./app migrate up \
#     && systemctl restart lana-api
set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}[deploy]${NC} $1"; }
fail() { echo -e "${RED}[deploy] ERROR:${NC} $1"; exit 1; }

log "Deploying lana-api (Go)..."
cd /root/lana-flowers-go || fail "lana-flowers-go not found"

git fetch origin
git reset --hard origin/main

go build -o app ./cmd/api || fail "go build failed"
systemctl restart lana-api || fail "systemctl restart lana-api failed"
log "lana-api restarted ✓"

log "All done!"
