# OpenClaw Setup

## Remote Facts

- run user: `admin`
- binary: `/home/admin/.local/share/pnpm/openclaw`
- gateway: `0.0.0.0:17291`
- workspace: `/home/admin/.openclaw/workspace`
- platform API for OpenClaw smoke: `http://127.0.0.1:18080`
- Telegram target: `6959476905`

## Configure Telegram Channel

```bash
sudo -u admin \
  HOME=/home/admin \
  XDG_CONFIG_HOME=/home/admin/.config \
  XDG_STATE_HOME=/home/admin/.local/state \
  XDG_DATA_HOME=/home/admin/.local/share \
  PATH=/usr/local/bin:/usr/bin:/bin \
  /home/admin/.local/share/pnpm/openclaw channels add --channel telegram --token "$TELEGRAM_BOT_TOKEN"
```

Verify:

```bash
sudo -u admin \
  HOME=/home/admin \
  XDG_CONFIG_HOME=/home/admin/.config \
  XDG_STATE_HOME=/home/admin/.local/state \
  XDG_DATA_HOME=/home/admin/.local/share \
  PATH=/usr/local/bin:/usr/bin:/bin \
  /home/admin/.local/share/pnpm/openclaw channels list
```

## Start platformd for OpenClaw smoke

`127.0.0.1:8080` is occupied by SearXNG on this ECS. For OpenClaw validation, run `platformd` on `127.0.0.1:18080`.

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
PATH=/usr/local/go/bin:/usr/bin:/bin go build -o ./bin/platformd ./cmd/platformd
./bin/platformd -config configs/demo-mstr-e2e.yaml -listen 127.0.0.1:18080
```

Smoke:

```bash
curl -fsS http://127.0.0.1:18080/health
curl -fsS http://127.0.0.1:18080/api/status
```

## Run OpenClaw Local Agent Against Platform

The OpenClaw workspace rule is pinned in `/home/admin/.openclaw/workspace/AGENTS.md` and `/home/admin/.openclaw/workspace/skills/quant-platform-operator/SKILL.md`.

```bash
sudo -u admin \
  HOME=/home/admin \
  XDG_CONFIG_HOME=/home/admin/.config \
  XDG_STATE_HOME=/home/admin/.local/state \
  XDG_DATA_HOME=/home/admin/.local/share \
  PATH=/usr/local/bin:/usr/bin:/bin \
  /home/admin/.local/share/pnpm/openclaw agent --local --message "platform status" --json
```

Deliver the result back to Telegram:

```bash
sudo -u admin \
  HOME=/home/admin \
  XDG_CONFIG_HOME=/home/admin/.config \
  XDG_STATE_HOME=/home/admin/.local/state \
  XDG_DATA_HOME=/home/admin/.local/share \
  PATH=/usr/local/bin:/usr/bin:/bin \
  /home/admin/.local/share/pnpm/openclaw agent --local \
    --message "platform status" \
    --json \
    --deliver \
    --reply-channel telegram \
    --reply-to 6959476905
```

## Safety Boundary

- OpenClaw reads platform state through `platformd` only.
- OpenClaw does not hold Bitget write credentials.
- Live flatten and any exchange write action stay behind platform approval and `execd`.
