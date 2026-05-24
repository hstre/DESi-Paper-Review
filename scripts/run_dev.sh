#!/usr/bin/env bash
# Run the DESi governance microservice and the Go web UI together.
# Ctrl-C stops both (the service is started in the background and killed
# on exit via a trap).
set -uo pipefail

DESI_PORT="${DESI_PORT:-8765}"
WEB_ADDR="${WEB_ADDR:-:8080}"
DESI_URL="http://127.0.0.1:${DESI_PORT}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Starting DESi governance service on ${DESI_URL} ..."
python3 service/desi_service.py --port "${DESI_PORT}" &
SVC=$!
cleanup() { echo; echo "stopping DESi service ..."; kill "${SVC}" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

for _ in $(seq 1 40); do
  curl -fsS "${DESI_URL}/health" >/dev/null 2>&1 && break
  sleep 0.25
done

echo "Building web UI ..."
( cd web && GOPROXY=off GOFLAGS=-mod=mod go build -o dpr-web . )

echo "Web UI on http://localhost${WEB_ADDR}  (DESi service: ${DESI_URL})"
./web/dpr-web -addr "${WEB_ADDR}" -desi "${DESI_URL}"
