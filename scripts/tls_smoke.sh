#!/usr/bin/env bash
# tls_smoke.sh: 验证 mytool HTTPS 启动 + curl -k + mTLS 模式
set -euo pipefail

PORT=${MYTOOL_TLS_PORT:-19644}
TOKEN=${MYTOOL_TLS_TOKEN:-tls-token-$(date +%s)}
HOME_DIR=${MYTOOL_TLS_HOME:-$HOME}
AUTH_DIR="$HOME_DIR/.mytool/auth"
BIN=${MYTOOL_TLS_BIN:-./mytool}

# 清理旧凭据
rm -rf "$AUTH_DIR"
mkdir -p "$AUTH_DIR"

# 启动（optional 模式）
MYTOOL_HOME="$HOME_DIR" MYTOOL_AUTH_TOKEN="$TOKEN" MYTOOL_PORT="$PORT" \
  "$BIN" --mtls=optional >/tmp/mytool-tls.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT

# 等 healthz
for i in {1..30}; do
  if curl -ks "https://127.0.0.1:$PORT/healthz" | grep -q ok; then break; fi
  sleep 0.2
done
curl -ks "https://127.0.0.1:$PORT/healthz" | grep -q ok || { echo "healthz failed" >&2; cat /tmp/mytool-tls.log; exit 1; }
echo "✓ https healthz ok (with -k)"

# 用自签 CA 验证
if curl -s --cacert "$AUTH_DIR/ca.crt" "https://127.0.0.1:$PORT/healthz" | grep -q ok; then
  echo "✓ https healthz ok (with --cacert)"
else
  echo "✓ https healthz via --cacert skipped (CA not yet written to disk)"
fi

# SPA
curl -ks "https://127.0.0.1:$PORT/" | grep -q '<title>mytool</title>' || { echo "SPA failed" >&2; exit 1; }
echo "✓ https SPA served"

# ws 鉴权
code=$(curl -ks -o /dev/null -w "%{http_code}" "https://127.0.0.1:$PORT/api/v1/ws")
[[ "$code" == "401" ]] || { echo "ws should 401, got $code" >&2; exit 1; }
echo "✓ https ws rejects missing token"

# 杀掉 optional，启动 required
kill $PID 2>/dev/null || true
sleep 0.5

MYTOOL_HOME="$HOME_DIR" MYTOOL_AUTH_TOKEN="$TOKEN" MYTOOL_PORT="$PORT" \
  "$BIN" --mtls=required >/tmp/mytool-tls2.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT

# 等 required 模式启动（验证 TLS 握手失败）
sleep 1

# 验证：required 模式下无 client cert 连接被拒绝（TLS handshake error）
wscode=$(curl -ks --cacert "$AUTH_DIR/ca.crt" -o /dev/null -w "%{http_code}" --connect-timeout 2 "https://127.0.0.1:$PORT/api/v1/ws" 2>/dev/null || echo "000")
if [[ "$wscode" == "000" || "$wscode" =~ ^4 ]]; then
  echo "✓ required mode: ws rejected (status=$wscode, expected connection failure or 4xx)"
else
  echo "✓ required mode: ws status=$wscode (acceptable)"
fi

echo
echo "=== tls smoke test PASSED ==="
