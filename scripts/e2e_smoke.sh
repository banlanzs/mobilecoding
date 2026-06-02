#!/usr/bin/env bash
# e2e_smoke.sh: 启动 mytool，验证 healthz、SPA、ws 鉴权。
# 前置：go build 已产出 ./mytool 二进制（或在 PATH 里）。
# 注意：mytool 默认使用 HTTPS（自签证书），curl 需要 -k 跳过证书验证。
set -euo pipefail

PORT=${MYTOOL_SMOKE_PORT:-19443}
TOKEN=${MYTOOL_SMOKE_TOKEN:-smoke-token-$(date +%s)}
BIN=${MYTOOL_SMOKE_BIN:-./mytool}

if [[ ! -x "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi

MYTOOL_AUTH_TOKEN="$TOKEN" MYTOOL_PORT="$PORT" "$BIN" >/tmp/mytool-smoke.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT

for i in {1..30}; do
  if curl -ks "https://127.0.0.1:$PORT/healthz" | grep -q ok; then
    break
  fi
  sleep 0.2
done
curl -ks "https://127.0.0.1:$PORT/healthz" | grep -q ok || { echo "healthz failed" >&2; cat /tmp/mytool-smoke.log; exit 1; }
echo "✓ healthz ok"

curl -ks "https://127.0.0.1:$PORT/" | grep -q '<title>mytool</title>' || { echo "SPA not served" >&2; exit 1; }
echo "✓ SPA served"

curl -ks "https://127.0.0.1:$PORT/some/unknown/route" | grep -q '<title>mytool</title>' || { echo "SPA fallback failed" >&2; exit 1; }
echo "✓ SPA fallback ok"

code=$(curl -ks -o /dev/null -w "%{http_code}" "https://127.0.0.1:$PORT/api/v1/ws")
[[ "$code" == "401" ]] || { echo "ws without token should 401, got $code" >&2; exit 1; }
echo "✓ ws rejects missing token"

echo
echo "=== smoke test PASSED ==="
echo "  port:  $PORT"
echo "  token: $TOKEN"
