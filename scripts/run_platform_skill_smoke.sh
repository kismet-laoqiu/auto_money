#!/usr/bin/env bash
set -euo pipefail

PATH=/usr/local/go/bin:/usr/bin:/bin
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PLATFORM_ADDR=${PLATFORM_ADDR:-http://127.0.0.1:18080}
RUN_ID=${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}
ARTIFACT_DIR=${ARTIFACT_DIR:-artifacts/platform/skill-smoke/$RUN_ID}
OPENCLAW_REPLY_TO=${OPENCLAW_REPLY_TO:-6959476905}
OPENCLAW_TO=${OPENCLAW_TO:-+15555550123}
OPENCLAW_BIN=${OPENCLAW_BIN:-/home/admin/.local/share/pnpm/openclaw}
LATEST_E2E_DIR=${LATEST_E2E_DIR:-$(find artifacts/platform/e2e -mindepth 1 -maxdepth 1 -type d 2>/dev/null | LC_ALL=C sort | tail -n 1)}

extract_openclaw_json() {
    local raw_file=$1
    local json_file=$2
    python3.11 - "$raw_file" "$json_file" <<'PYJSON'
import json
import sys
from pathlib import Path

raw = Path(sys.argv[1]).read_text()
decoder = json.JSONDecoder()
for index, char in enumerate(raw):
    if char != '{':
        continue
    try:
        payload, _ = decoder.raw_decode(raw[index:])
    except json.JSONDecodeError:
        continue
    Path(sys.argv[2]).write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n")
    raise SystemExit(0)
raise SystemExit(f"failed to extract openclaw json from {sys.argv[1]}")
PYJSON
}

run_openclaw_json() {
    local json_file=$1
    shift
    local raw_file="${json_file%.json}.raw.log"
    "$@" >"$raw_file" 2>&1
    extract_openclaw_json "$raw_file" "$json_file"
}

cd "$ROOT_DIR"
mkdir -p "$ARTIFACT_DIR"
curl -fsS "$PLATFORM_ADDR/health" >"$ARTIFACT_DIR/platform-health.json"
if [[ -z "$LATEST_E2E_DIR" || ! -f "$LATEST_E2E_DIR/status.json" || ! -f "$LATEST_E2E_DIR/strategy-versions.json" ]]; then
    echo "missing latest e2e artifacts under artifacts/platform/e2e" >&2
    exit 1
fi

/usr/bin/codex exec \
    --dangerously-bypass-approvals-and-sandbox \
    -C "$ROOT_DIR" \
    --output-last-message "$ARTIFACT_DIR/codex.txt" \
    "Use quant-platform-operator. Inspect platform status and strategy versions. Do not edit files."

printf "%s\n" "Read .agents/skills/quant-platform-operator/SKILL.md, then read $LATEST_E2E_DIR/status.json and $LATEST_E2E_DIR/strategy-versions.json. Summarize current platform last_seq and newest strategy version only. Do not run commands. Do not edit files." | \
    /usr/bin/claude -p \
    --add-dir "$ROOT_DIR" \
    >"$ARTIFACT_DIR/claude.txt"

run_openclaw_json "$ARTIFACT_DIR/openclaw.json" \
    sudo -u admin \
        HOME=/home/admin \
        XDG_CONFIG_HOME=/home/admin/.config \
        XDG_STATE_HOME=/home/admin/.local/state \
        XDG_DATA_HOME=/home/admin/.local/share \
        PATH=/usr/local/bin:/usr/bin:/bin \
        "$OPENCLAW_BIN" agent --local --message "platform status" --json --to "$OPENCLAW_TO"

run_openclaw_json "$ARTIFACT_DIR/openclaw-deliver.json" \
    sudo -u admin \
        HOME=/home/admin \
        XDG_CONFIG_HOME=/home/admin/.config \
        XDG_STATE_HOME=/home/admin/.local/state \
        XDG_DATA_HOME=/home/admin/.local/share \
        PATH=/usr/local/bin:/usr/bin:/bin \
        "$OPENCLAW_BIN" agent --local \
            --message "platform status" \
            --json \
            --deliver \
            --reply-channel telegram \
            --reply-to "$OPENCLAW_REPLY_TO" \
            --to "$OPENCLAW_TO"

ARTIFACT_DIR="$ARTIFACT_DIR" python3.11 - <<'PYSUM' >"$ARTIFACT_DIR/summary.json"
import json
import os
import sys
from pathlib import Path

root = Path(os.environ['ARTIFACT_DIR'])
summary = {
    'artifact_dir': str(root),
    'steps': ['codex', 'claude', 'openclaw-local', 'openclaw-deliver-telegram'],
    'files': {
        'codex': str(root / 'codex.txt'),
        'claude': str(root / 'claude.txt'),
        'openclaw': str(root / 'openclaw.json'),
        'openclaw_raw': str(root / 'openclaw.raw.log'),
        'openclaw_deliver': str(root / 'openclaw-deliver.json'),
        'openclaw_deliver_raw': str(root / 'openclaw-deliver.raw.log'),
    },
}
json.dump(summary, fp=sys.stdout, indent=2)
print()
PYSUM

echo "platform skill smoke ok: $ARTIFACT_DIR"
