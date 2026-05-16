#!/bin/bash
# Деплой бэка на VPS: git pull → пересборка → рестарт сервиса.
#
# Запускать НА СЕРВЕРЕ из /root/lana-flowers:
#   ./deploy.sh
#
# Или с локальной машины одной строкой:
#   ssh root@64.226.107.161 'cd /root/lana-flowers && ./deploy.sh'
#
# Сборка занимает ~6 мин (458MB RAM + 2GB swap, поэтому терпимо).
# Если упадёт по OOM — увеличить swap, см. ниже.
#
# Фронт деплоится отдельно через `vercel --prod` с локальной машины.

set -euo pipefail

cd "$(dirname "$0")"

echo "==> git pull"
git pull --ff-only

echo "==> go build (~6 мин на 458MB RAM + swap)"
cd lana-flowers-go
time go build -o app ./cmd/api

echo "==> restart"
systemctl restart lana-api
sleep 1
systemctl is-active lana-api
journalctl -u lana-api -n 3 --no-pager
