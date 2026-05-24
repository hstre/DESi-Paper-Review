#!/usr/bin/env bash
# Full health gate: start the DESi governance microservice, then run the
# Python doctor (with --service-url) and the Go web self-check against it.
# Exits non-zero if any check fails. The service is stopped on exit.
set -uo pipefail

DESI_PORT="${DESI_PORT:-8765}"
DESI_URL="http://127.0.0.1:${DESI_PORT}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

python3 service/desi_service.py --port "${DESI_PORT}" >/tmp/desi_service.check.log 2>&1 &
SVC=$!
cleanup() { kill "${SVC}" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

up=0
for _ in $(seq 1 40); do
  if curl -fsS "${DESI_URL}/health" >/dev/null 2>&1; then up=1; break; fi
  sleep 0.25
done
if [ "${up}" -ne 1 ]; then
  echo "[FAIL] DESi service did not become healthy"
  cat /tmp/desi_service.check.log
  exit 1
fi

rc=0
echo "### Python doctor"
desi-paper-review doctor --service-url "${DESI_URL}" || rc=1
echo
echo "### Go web self-check"
( cd web && GOPROXY=off GOFLAGS=-mod=mod go build -o dpr-web . ) || rc=1
./web/dpr-web -check -desi "${DESI_URL}" || rc=1

echo
if [ "${rc}" -eq 0 ]; then
  echo "ALL_CHECKS_PASSED"
else
  echo "CHECKS_FAILED"
fi
exit "${rc}"
