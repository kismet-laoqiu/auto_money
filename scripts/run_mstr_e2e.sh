#!/usr/bin/env bash
set -euo pipefail

PATH=/usr/local/go/bin:/usr/bin:/bin
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SOURCE_CONFIG_PATH=${1:-configs/demo-mstr-e2e.yaml}
REST_BASE_URL=${REST_BASE_URL:-https://api.bitget.com}
SYMBOL=${SYMBOL:-MSTRUSDT}
PRODUCT_TYPE=${PRODUCT_TYPE:-USDT-FUTURES}
MARGIN_MODE=${MARGIN_MODE:-isolated}
LEVERAGE=${LEVERAGE:-3}
DIRECTION=${DIRECTION:-long}
SIZE=${SIZE:-}
SMOKE_SECONDS=${SMOKE_SECONDS:-5}
RUN_ID=${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}
ARTIFACT_DIR=${ARTIFACT_DIR:-artifacts/mstr-e2e/$RUN_ID}
CONFIG_PATH="$ARTIFACT_DIR/runtime-config.yaml"
STATE_DB_PATH="$ARTIFACT_DIR/runtime-state.db"
MARKETD_LOG="$ARTIFACT_DIR/marketd-config-smoke.log"
TRADERD_LOG="$ARTIFACT_DIR/traderd-config-smoke.log"
RULES_JSON="$ARTIFACT_DIR/contract-rules.json"
TICKER_JSON="$ARTIFACT_DIR/ticker.json"
SUMMARY_JSON="$ARTIFACT_DIR/preflight-summary.json"
SIZE_FILE="$ARTIFACT_DIR/effective-size.txt"

require_env() {
	local name=$1
	if [[ -z "${!name:-}" ]]; then
		echo "missing required env: $name" >&2
		exit 1
	fi
}

run_smoke() {
	local name=$1
	local target=$2
	local logfile=$3
	local start_epoch elapsed status min_elapsed

	start_epoch=$(date +%s)
	set +e
	timeout --preserve-status --signal=INT --kill-after=2s "${SMOKE_SECONDS}s" "$target" -config "$CONFIG_PATH" >"$logfile" 2>&1
	status=$?
	set -e
	elapsed=$(( $(date +%s) - start_epoch ))
	min_elapsed=$(( SMOKE_SECONDS - 1 ))
	if [[ $min_elapsed -lt 1 ]]; then
		min_elapsed=1
	fi
	if [[ $status -ne 0 ]]; then
		cat "$logfile" >&2
		echo "$name config smoke failed with exit code $status" >&2
		exit "$status"
	fi
	if [[ $elapsed -lt $min_elapsed ]]; then
		cat "$logfile" >&2
		echo "$name config smoke exited too early after ${elapsed}s" >&2
		exit 1
	fi
}

cd "$ROOT_DIR"
mkdir -p "$ARTIFACT_DIR"

require_env BITGET_API_KEY
require_env BITGET_API_SECRET
require_env BITGET_PASSPHRASE

curl --silent --show-error --fail \
	"$REST_BASE_URL/api/v2/mix/market/contracts?productType=$PRODUCT_TYPE" \
	-o "$RULES_JSON"

curl --silent --show-error --fail \
	"$REST_BASE_URL/api/v2/mix/market/ticker?symbol=$SYMBOL&productType=$PRODUCT_TYPE" \
	-o "$TICKER_JSON"

RULES_JSON="$RULES_JSON" \
TICKER_JSON="$TICKER_JSON" \
SYMBOL="$SYMBOL" \
PRODUCT_TYPE="$PRODUCT_TYPE" \
MARGIN_MODE="$MARGIN_MODE" \
LEVERAGE="$LEVERAGE" \
DIRECTION="$DIRECTION" \
SIZE="$SIZE" \
SOURCE_CONFIG_PATH="$SOURCE_CONFIG_PATH" \
CONFIG_PATH="$CONFIG_PATH" \
STATE_DB_PATH="$STATE_DB_PATH" \
SUMMARY_JSON="$SUMMARY_JSON" \
SIZE_FILE="$SIZE_FILE" \
python3.11 - <<'PY'
import json
import math
import os
from pathlib import Path

rules_path = Path(os.environ["RULES_JSON"])
ticker_path = Path(os.environ["TICKER_JSON"])
summary_path = Path(os.environ["SUMMARY_JSON"])
size_path = Path(os.environ["SIZE_FILE"])
symbol = os.environ["SYMBOL"]
payload = json.loads(rules_path.read_text(encoding="utf-8"))
items = payload.get("data") or []
match = None
for item in items:
    if item.get("symbol") == symbol:
        match = item
        break
if match is None:
    raise SystemExit(f"symbol not found in contract rules: {symbol}")
min_trade_num = float(match["minTradeNum"])
size_multiplier = float(match.get("sizeMultiplier") or "0") or min_trade_num or 0.01
min_trade_usdt = float(match.get("minTradeUSDT") or "0")
max_lever_raw = match.get("maxLeverage") or match.get("maxLever") or "0"
max_leverage = int(max_lever_raw)
if abs(min_trade_num - 0.01) > 1e-9:
    raise SystemExit(f"unexpected minTradeNum for {symbol}: {min_trade_num}")
if max_leverage < int(os.environ["LEVERAGE"]):
    raise SystemExit(f"unexpected max leverage for {symbol}: {max_leverage}")
ticker_payload = json.loads(ticker_path.read_text(encoding="utf-8"))
ticker_items = ticker_payload.get("data") or []
if not ticker_items:
    raise SystemExit(f"ticker not found for {symbol}")
last_price_raw = str(ticker_items[0].get("lastPr") or ticker_items[0].get("markPrice") or "").strip()
if not last_price_raw:
    raise SystemExit(f"ticker price missing for {symbol}")
last_price = float(last_price_raw)
requested_size_raw = os.environ.get("SIZE", "").strip()
requested_size = float(requested_size_raw) if requested_size_raw else 0.0

def ceil_to_step(value: float, step: float) -> float:
    return math.ceil((value / step) - 1e-12) * step

effective_size = ceil_to_step(max(requested_size, min_trade_num), size_multiplier)
if min_trade_usdt > 0 and last_price > 0:
    effective_size = max(
        effective_size,
        ceil_to_step(min_trade_usdt / last_price, size_multiplier),
    )
    while effective_size * last_price + 1e-9 < min_trade_usdt:
        effective_size += size_multiplier
effective_size = round(effective_size, 8)
effective_size_text = f"{effective_size:.8f}".rstrip("0").rstrip(".")
summary = {
    "symbol": symbol,
    "product_type": os.environ["PRODUCT_TYPE"],
    "margin_mode": os.environ["MARGIN_MODE"],
    "leverage": int(os.environ["LEVERAGE"]),
    "direction": os.environ["DIRECTION"],
    "requested_size": requested_size,
    "size": effective_size,
    "last_price": last_price,
    "config_path": os.environ["SOURCE_CONFIG_PATH"],
    "runtime_config_path": os.environ["CONFIG_PATH"],
    "state_db_path": os.environ["STATE_DB_PATH"],
    "contract_rule": {
        "symbolStatus": match.get("symbolStatus"),
        "minTradeNum": min_trade_num,
        "sizeMultiplier": size_multiplier,
        "minTradeUSDT": min_trade_usdt,
        "maxLever": max_leverage,
        "pricePlace": match.get("pricePlace"),
        "volumePlace": match.get("volumePlace"),
    },
    "credential_presence": {
        "BITGET_API_KEY": True,
        "BITGET_API_SECRET": True,
        "BITGET_PASSPHRASE": True,
    },
}
summary_path.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
size_path.write_text(effective_size_text + "\n", encoding="utf-8")
print(json.dumps(summary, indent=2))
PY

if ! grep -q '^    state_db_path:' "$SOURCE_CONFIG_PATH"; then
	echo "missing runtime.state_db_path in $SOURCE_CONFIG_PATH" >&2
	exit 1
fi
sed "s|^    state_db_path: .*|    state_db_path: $STATE_DB_PATH|" "$SOURCE_CONFIG_PATH" >"$CONFIG_PATH"
SIZE=$(cat "$SIZE_FILE")

go build -o /tmp/quantlab-marketd ./cmd/marketd
go build -o /tmp/quantlab-traderd ./cmd/traderd

run_smoke marketd /tmp/quantlab-marketd "$MARKETD_LOG"
run_smoke traderd /tmp/quantlab-traderd "$TRADERD_LOG"

echo "preflight ok: $SUMMARY_JSON"
