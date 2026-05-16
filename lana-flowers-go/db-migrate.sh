#!/bin/bash
# Локальный запуск миграций через SSH-туннель к prod-БД.
# Использование: ./db-migrate.sh up | down | version | force N | reset
#
# Поднимает туннель localhost:15432 -> server:5432 если ещё не поднят,
# подменяет host:port в DATABASE_URL на туннельный и запускает migrate.
# SSH-ключ должен быть уже залит на сервер.
set -e

TUNNEL_PORT=15432
SERVER_HOST="${LF_SERVER_HOST:?LF_SERVER_HOST not set in env or .env}"
SERVER_USER="${LF_SERVER_USER:-root}"

if ! ssh -o BatchMode=yes -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_HOST} true 2>/dev/null; then
    echo "[db] SSH-key not set up. Run: ssh-copy-id ${SERVER_USER}@${SERVER_HOST}" >&2
    exit 1
fi

if command -v ss >/dev/null && ss -ltn 2>/dev/null | grep -q ":${TUNNEL_PORT} "; then
    :
elif command -v netstat >/dev/null && netstat -an 2>/dev/null | grep -q "127.0.0.1.${TUNNEL_PORT}.*LISTEN"; then
    :
else
    echo "[db] starting SSH tunnel on localhost:${TUNNEL_PORT}..."
    ssh -f -N -L ${TUNNEL_PORT}:localhost:5432 -o ExitOnForwardFailure=yes ${SERVER_USER}@${SERVER_HOST}
fi

# Подменяем host:port в DATABASE_URL на туннельный.
DB_URL=$(grep '^DATABASE_URL' .env | sed 's/^DATABASE_URL=//' | tr -d '"' | sed -E "s|@[^/]+|@localhost:${TUNNEL_PORT}|")

echo "[db] migrate $@"
DATABASE_URL="${DB_URL}" go run ./cmd/api migrate "$@"
