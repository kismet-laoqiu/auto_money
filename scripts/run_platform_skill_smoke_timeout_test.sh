#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
export OPENCLAW_TIMEOUT_SECONDS=1

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

source "$ROOT_DIR/scripts/run_platform_skill_smoke.sh"

json_file="$tmpdir/openclaw.json"
start_epoch=$(date +%s)

run_openclaw_json "$json_file" \
    bash -lc 'printf "{\"payloads\":[{\"text\":\"ok\"}],\"meta\":{\"stopReason\":\"stop\"}}\n"; sleep 5'

elapsed=$(( $(date +%s) - start_epoch ))

if [[ ! -f "$json_file" ]]; then
    echo "json output missing: $json_file" >&2
    exit 1
fi

JSON_FILE="$json_file" python3.11 - <<'PY'
import json
import os
from pathlib import Path

payload = json.loads(Path(os.environ["JSON_FILE"]).read_text())
text = payload["payloads"][0]["text"]
if text != "ok":
    raise SystemExit(f"unexpected payload text: {text!r}")
PY

if [[ $elapsed -ge 4 ]]; then
    echo "run_openclaw_json took too long: ${elapsed}s" >&2
    exit 1
fi

echo "run_platform_skill_smoke timeout test ok"
