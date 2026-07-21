#!/usr/bin/env bash
set -euo pipefail
cd /root/relayQ-test

echo "[1/6] fetch latest main"
git fetch origin
git checkout main
git reset --hard origin/main

echo "[2/6] ensure frontend deps locally cached in repo"
cd frontend
corepack enable >/dev/null 2>&1 || true
corepack prepare pnpm@11.10.0 --activate >/dev/null 2>&1 || true
export PATH="$HOME/.local/share/pnpm:$PATH"
pnpm install --no-frozen-lockfile --ignore-scripts

echo "[3/6] build frontend only"
pnpm run build
cd ..

echo "[4/6] rebuild backend image with embedded dist"
docker build -t sub2api:local -f deploy/Dockerfile .

echo "[5/6] restart service"
cd deploy
docker compose up -d --force-recreate sub2api

echo "[6/6] verify"
docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' | grep sub2api || true
curl -sS http://127.0.0.1:8080/health || true
