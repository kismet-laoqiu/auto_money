#!/usr/bin/env bash
set -euo pipefail

PATH=/usr/local/go/bin:/usr/bin:/bin
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CHECK_CONFIG_PATH=${1:-configs/baseline.yaml}
BASELINE_CONFIG_PATH=configs/baseline.yaml
SMOKE_SECONDS=${SMOKE_SECONDS:-5}
LAB_BIN=/tmp/quantlab-lab
STREAM_BIN=/tmp/quantlab-stream
MARKETD_BIN=/tmp/quantlab-marketd
TRADERD_BIN=/tmp/quantlab-traderd
AGENTD_BIN=/tmp/quantlab-agentd
STREAM_LOG=/tmp/quantlab-stream-smoke.log
MARKETD_LOG=/tmp/quantlab-marketd-smoke.log
TRADERD_LOG=/tmp/quantlab-traderd-smoke.log
REPLAY_JSON=/tmp/quantlab-replay.json

cd "$ROOT_DIR"

run_smoke() {
	local name=$1
	local binary=$2
	local config_path=$3
	local logfile=$4
	local start_epoch elapsed status min_elapsed

	start_epoch=$(date +%s)
	set +e
	timeout --preserve-status --signal=INT --kill-after=2s "${SMOKE_SECONDS}s" "$binary" -config "$config_path" > "$logfile" 2>&1
	status=$?
	set -e
	elapsed=$(( $(date +%s) - start_epoch ))
	min_elapsed=$(( SMOKE_SECONDS - 1 ))
	if [[ $min_elapsed -lt 1 ]]; then
		min_elapsed=1
	fi
	if [[ $status -ne 0 ]]; then
		cat "$logfile" >&2
		echo "$name smoke failed with exit code $status" >&2
		exit $status
	fi
	if [[ $elapsed -lt $min_elapsed ]]; then
		cat "$logfile" >&2
		echo "$name smoke exited too early after ${elapsed}s" >&2
		exit 1
	fi
}

runtime_config_name=$(basename "$CHECK_CONFIG_PATH")
is_live_runtime=0
	case "$runtime_config_name" in
		live.yaml)
		is_live_runtime=1
		;;
esac

go test ./...
go build -o "$LAB_BIN" ./cmd/lab
go build -o "$STREAM_BIN" ./cmd/stream
go build -o "$MARKETD_BIN" ./cmd/marketd
go build -o "$TRADERD_BIN" ./cmd/traderd
go build -o "$AGENTD_BIN" ./cmd/agentd

"$LAB_BIN" eval -config "$BASELINE_CONFIG_PATH" > metrics.json
if [[ "$is_live_runtime" == "1" ]]; then
	"$LAB_BIN" replay -config "$CHECK_CONFIG_PATH" > "$REPLAY_JSON"
else
	"$LAB_BIN" replay -config "$BASELINE_CONFIG_PATH" > "$REPLAY_JSON"
fi

if [[ "$is_live_runtime" == "1" ]]; then
	run_smoke marketd "$MARKETD_BIN" "$CHECK_CONFIG_PATH" "$MARKETD_LOG"
	run_smoke traderd "$TRADERD_BIN" "$CHECK_CONFIG_PATH" "$TRADERD_LOG"
else
	run_smoke stream "$STREAM_BIN" "$BASELINE_CONFIG_PATH" "$STREAM_LOG"
fi

CHECK_CONFIG_PATH="$CHECK_CONFIG_PATH" python3.11 - <<'PY'
import json
import os
from pathlib import Path

metrics = json.loads(Path('metrics.json').read_text(encoding='utf-8'))
replay = json.loads(Path('/tmp/quantlab-replay.json').read_text(encoding='utf-8'))
robust = metrics.get('robust', {})
runtime_config_name = Path(os.environ['CHECK_CONFIG_PATH']).name
print(f"OBJECTIVE_SCORE={metrics.get('objective_score', 0):.12f}")
print(f"FINAL_SCORE={metrics.get('final_score', 0):.12f}")
print(f"MEDIAN_OBJECTIVE_SCORE={robust.get('median_objective_score', 0):.12f}")
print(f"MIN_BUCKET_MEAN={robust.get('min_bucket_mean', 0):.12f}")
print(f"POSITIVE_OOS_RETURN_RATIO={robust.get('positive_oos_return_ratio', 0):.12f}")
print(f"REPLAY_EVENT_COUNT={replay.get('event_count', 0)}")
print(f"REPLAY_COMMAND_COUNT={replay.get('command_count', 0)}")
print(f"REPLAY_CANDIDATE_COUNT={replay.get('candidate_count', 0)}")
if runtime_config_name == 'live.yaml':
    print(f"RUNTIME_CONFIG={runtime_config_name}")
    print(f"RUNTIME_SMOKE_SECONDS={os.environ.get('SMOKE_SECONDS', '5')}")
for bucket, value in sorted(robust.get('bucket_means', {}).items()):
    print(f"BUCKET_{bucket.upper()}={value:.12f}")
PY
