#!/usr/bin/env bash
set -euo pipefail

PATH=/usr/local/go/bin:/usr/bin:/bin
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PLATFORM_ADDR=${PLATFORM_ADDR:-http://127.0.0.1:18080}
WAREHOUSE_CONFIG=${WAREHOUSE_CONFIG:-configs/platform/warehouse.yaml}
BUNDLE_CONFIG=${BUNDLE_CONFIG:-configs/demo-mstr-bundle.yaml}
PROMOTION_VERSION=${PROMOTION_VERSION:-v0.1.2}
RUN_ID=${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}
ARTIFACT_DIR=${ARTIFACT_DIR:-artifacts/platform/e2e/$RUN_ID}
SYNC_ARTIFACT_ROOT="$ARTIFACT_DIR/historical-sync"
EXPORT_ARTIFACT_ROOT="$ARTIFACT_DIR/export"
MCP_HOST=${MCP_HOST:-127.0.0.1}
MCP_PORT=${MCP_PORT:-18082}
SYMBOLS=${SYMBOLS:-MSTRUSDT}
INTERVALS=${INTERVALS:-1m,5m,15m,1h,4h,1d}

ensure_bin() {
    local target=$1
    local pkg=$2
    if [[ ! -x "$target" ]]; then
        mkdir -p "$(dirname "$target")"
        go build -o "$target" "$pkg"
    fi
}

run_json() {
    local output=$1
    shift
    "$@" >"$output"
}

cd "$ROOT_DIR"
mkdir -p "$ARTIFACT_DIR"
ensure_bin ./bin/platformctl ./cmd/platformctl

curl -fsS "$PLATFORM_ADDR/health" >"$ARTIFACT_DIR/platform-health.json"
run_json "$ARTIFACT_DIR/warehouse-health.json" ./bin/platformctl warehouse health -config "$WAREHOUSE_CONFIG"
run_json "$ARTIFACT_DIR/historical-sync.json" ./bin/platformctl historical sync \
    -warehouse-config "$WAREHOUSE_CONFIG" \
    -provider bitget \
    -product-type USDT-FUTURES \
    -symbols "$SYMBOLS" \
    -intervals "$INTERVALS" \
    -limit 20 \
    -artifact-root "$SYNC_ARTIFACT_ROOT"
run_json "$ARTIFACT_DIR/aggregate.json" ./bin/platformctl aggregate run \
    -warehouse-config "$WAREHOUSE_CONFIG" \
    -provider bitget \
    -product-type USDT-FUTURES \
    -symbols "$SYMBOLS" \
    -intervals 5m,15m,1h,4h,1d
run_json "$ARTIFACT_DIR/export.json" ./bin/platformctl export parquet \
    -warehouse-config "$WAREHOUSE_CONFIG" \
    -provider bitget \
    -symbols "$SYMBOLS" \
    -intervals "$INTERVALS" \
    -artifact-root "$EXPORT_ARTIFACT_ROOT"
run_json "$ARTIFACT_DIR/status.json" ./bin/platformctl status -addr "$PLATFORM_ADDR"
run_json "$ARTIFACT_DIR/strategy-versions.json" ./bin/platformctl strategy versions -addr "$PLATFORM_ADDR" -strategy mstr-wave-fib
run_json "$ARTIFACT_DIR/backtest.json" ./bin/platformctl backtest run -addr "$PLATFORM_ADDR" -config "$BUNDLE_CONFIG"
run_json "$ARTIFACT_DIR/promotion-request.json" ./bin/platformctl promotion request -addr "$PLATFORM_ADDR" -strategy mstr-wave-fib -version "$PROMOTION_VERSION" -config "$BUNDLE_CONFIG"

MCP_HOST="$MCP_HOST" MCP_PORT="$MCP_PORT" python3.11 - <<'PYMCP' >"$ARTIFACT_DIR/mcp-tools-list.json"
import json
import os
import socket
import sys

host = os.environ.get('MCP_HOST', '127.0.0.1')
port = int(os.environ.get('MCP_PORT', '18082'))
messages = [
    {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "platform-e2e", "version": "1.0.0"}}},
    {"jsonrpc": "2.0", "method": "notifications/initialized"},
    {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}},
]
responses = []
with socket.create_connection((host, port), timeout=5) as sock:
    for message in messages:
        sock.sendall((json.dumps(message) + "\n").encode())
        if 'id' not in message:
            continue
        data = b''
        while not data.endswith(b'\n'):
            chunk = sock.recv(65536)
            if not chunk:
                raise SystemExit('mcp socket closed unexpectedly')
            data += chunk
        responses.append(json.loads(data.decode()))
json.dump(responses, fp=sys.stdout, indent=2)
print()
PYMCP

ARTIFACT_DIR="$ARTIFACT_DIR" PLATFORM_ADDR="$PLATFORM_ADDR" python3.11 - <<'PYSUM' >"$ARTIFACT_DIR/summary.json"
import json
import os
import sys
from pathlib import Path

root = Path(os.environ['ARTIFACT_DIR'])
backtest = json.loads((root / 'backtest.json').read_text())
promotion = json.loads((root / 'promotion-request.json').read_text())
summary = {
    'artifact_dir': str(root),
    'platform_addr': os.environ['PLATFORM_ADDR'],
    'steps': [
        'warehouse health',
        'historical sync',
        'aggregate',
        'export parquet',
        'platform status',
        'strategy versions',
        'backtest run',
        'promotion request',
        'mcp tools list',
    ],
    'backtest': {
        'final_score': backtest.get('final_score'),
        'objective_score': backtest.get('objective_score'),
    },
    'promotion': {
        'id': promotion.get('id'),
        'state': promotion.get('state'),
        'version': promotion.get('version'),
    },
}
json.dump(summary, fp=sys.stdout, indent=2)
print()
PYSUM

echo "platform e2e ok: $ARTIFACT_DIR"
