#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

docker compose up -d db

echo "Waiting for Postgres to be healthy..."
for i in $(seq 1 60); do
  status="$(docker compose ps --format json db | tr -d '\n' || true)"
  if echo "$status" | rg -q '"Health":"healthy"'; then
    break
  fi
  sleep 1
done

export DATABASE_URL="${DATABASE_URL:-postgres://pokemon:pokemon@localhost:5432/pokemon_web?sslmode=disable}"
export PORT="${PORT:-3030}"
export FRONTEND_DIR="${FRONTEND_DIR:-$(pwd)/frontend}"
export ROMS_DIR="${ROMS_DIR:-$(pwd)/roms}"

echo "Starting backend on http://localhost:${PORT}"
echo "DATABASE_URL=${DATABASE_URL}"
echo "FRONTEND_DIR=${FRONTEND_DIR}"
echo "ROMS_DIR=${ROMS_DIR}"
exec ./backend/pokemon-web
