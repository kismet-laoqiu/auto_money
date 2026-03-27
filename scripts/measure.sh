#!/usr/bin/env bash
set -euo pipefail

PATH=/usr/local/go/bin:$PATH
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CONFIG_PATH=${1:-configs/baseline.yaml}
LAB_BIN=/tmp/quantlab-lab
STREAM_BIN=/tmp/quantlab-stream
STREAM_LOG=/tmp/quantlab-stream-smoke.log
REPLAY_JSON=/tmp/quantlab-replay.json

cd "$ROOT_DIR"

go test ./...
go build -o "$LAB_BIN" ./cmd/lab
go build -o "$STREAM_BIN" ./cmd/stream

"$LAB_BIN" eval -config "$CONFIG_PATH" > metrics.json
"$LAB_BIN" replay -config "$CONFIG_PATH" > "$REPLAY_JSON"

start_epoch=$(date +%s)
set +e
timeout --preserve-status --signal=INT --kill-after=2s 5s "$STREAM_BIN" -config "$CONFIG_PATH" > "$STREAM_LOG" 2>&1
stream_status=$?
set -e
elapsed=$(( $(date +%s) - start_epoch ))
if [[ $stream_status -ne 0 ]]; then
	cat "$STREAM_LOG" >&2
	echo "stream smoke failed with exit code $stream_status" >&2
	exit $stream_status
fi
if [[ $elapsed -lt 4 ]]; then
	cat "$STREAM_LOG" >&2
	echo "stream smoke exited too early after ${elapsed}s" >&2
	exit 1
fi

python3.11 - <<'PY'
import json
from pathlib import Path

metrics = json.loads(Path('metrics.json').read_text(encoding='utf-8'))
replay = json.loads(Path('/tmp/quantlab-replay.json').read_text(encoding='utf-8'))
robust = metrics.get('robust', {})
print(f"OBJECTIVE_SCORE={metrics.get('objective_score', 0):.12f}")
print(f"FINAL_SCORE={metrics.get('final_score', 0):.12f}")
print(f"MEDIAN_OBJECTIVE_SCORE={robust.get('median_objective_score', 0):.12f}")
print(f"MIN_BUCKET_MEAN={robust.get('min_bucket_mean', 0):.12f}")
print(f"POSITIVE_OOS_RETURN_RATIO={robust.get('positive_oos_return_ratio', 0):.12f}")
print(f"REPLAY_EVENT_COUNT={replay.get('event_count', 0)}")
print(f"REPLAY_COMMAND_COUNT={replay.get('command_count', 0)}")
print(f"REPLAY_CANDIDATE_COUNT={replay.get('candidate_count', 0)}")
for bucket, value in sorted(robust.get('bucket_means', {}).items()):
    print(f"BUCKET_{bucket.upper()}={value:.12f}")
PY
