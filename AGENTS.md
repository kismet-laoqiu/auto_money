# ECS Workspace Guidelines

## 目录定位

该目录是 `47.250.138.143` 这台 ECS 的入口说明目录，不是本机开发目录。

- 本目录下的所有实际操作都必须先登录目标 ECS 再执行。
- 本目录下的所有 inspect、edit、build、test、debug、deploy、review、编码都必须在远端服务器上完成。
- 不得把“先在本机改，再同步到服务器”当成默认 workflow。
- `openclaw 密钥.md` 只用于提供连接信息，不应被当成本地开发工作区。

## 已验证的连接信息

以下信息来自 [`openclaw 密钥.md`](./openclaw%20密钥.md) 与实际 SSH 验证：

- SSH user: `root`
- Public IP: `47.250.138.143`
- Private IP: `172.19.52.41`
- Hostname: `iZ8psefacx3fh2dayewzbeZ`
- SSH port: `22`
- Private key fingerprint: `SHA256:ZLHtQYeGOkk+AYnULZqnfmDORNgKz49mCguJULniY/E`

## 登录方式

推荐方式是从 [`openclaw 密钥.md`](./openclaw%20密钥.md) 临时提取私钥，设置严格权限，然后用 `root` 直连：

```sh
workdir=$(mktemp -d /tmp/ecs-ssh.XXXXXX)
keyfile="$workdir/id_rsa"
awk 'BEGIN{flag=0} /-----BEGIN RSA PRIVATE KEY-----/{flag=1} flag{gsub(/^  /, ""); print} /-----END RSA PRIVATE KEY-----/{flag=0}' \
  './openclaw 密钥.md' > "$keyfile"
chmod 600 "$keyfile"
ssh -i "$keyfile" \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile="$workdir/known_hosts" \
  root@47.250.138.143
```

连接后应先做最小验证：

```sh
id -un
hostname
hostname -I
```

预期结果：

- 当前用户是 `root`
- 主机名包含 `iZ8psefacx3fh2dayewzbeZ`
- 内网地址包含 `172.19.52.41`

退出后应删除临时密钥文件，不要把私钥长期落盘到仓库目录。

## 远端优先规则

从这个目录出发处理任何任务时，默认顺序必须是：

1. 先读取 [`openclaw 密钥.md`](./openclaw%20密钥.md)。
2. 先登录 `root@47.250.138.143`。
3. 在远端确认目标目录、代码仓库、运行环境和依赖。
4. 在远端执行 inspect、edit、build、test、debug、deploy。
5. 仅把本目录当成连接说明和操作约束，不把它当成真实执行环境。

当前已验证的真实代码仓库路径是：

- `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan`

当前已验证的远端分支是：

- `autoresearch/20260328-all-plan`

## 远端写文件协议

当任务要求“必须在远端改代码”时，不要再尝试长 heredoc 或大块 inline patch 直接穿透 `exec_command -> ssh -> remote shell`。

- 高风险路径：长 heredoc、inline `git apply`、内联大段 `python -c`、需要多层引号逃逸的远端 patch。
- 已复现的失败模式：`corrupt patch`、本地变量提前展开、远端 shell quoting 污染。
- 优先协议：本地临时文件 + `scp` 到远端目标路径，或本地临时目录 + `rsync/scp` 到远端工作树。
- 如果只是读远端状态，可以直接 `ssh 'bash -lc ...'`；如果要改文件，优先走“拉到本地临时路径 → 本地 `apply_patch` → 推回远端”的稳定闭环。

## 远端 Git / Push 规则

当前远端仓库的 `origin` 已验证为：

- `git@github.com:kismet-laoqiu/auto_money.git`

当前远端主机上的 GitHub 专用私钥已验证存在且可用：

- `~/.ssh/id_ed25519_auto_money`

在这台 ECS 上，裸跑 `ssh -T git@github.com` 可能会因为默认身份选择错误而失败，即使专用 key 本身可用。因此：

- 不要把 `Permission denied (publickey)` 直接解读成“远端没有 GitHub key”。
- 先单独验证专用 key：

```sh
ssh -T -i ~/.ssh/id_ed25519_auto_money \
  -o IdentitiesOnly=yes \
  -o StrictHostKeyChecking=no \
  git@github.com
```

- 预期输出包含：`Hi kismet-laoqiu! You've successfully authenticated...`
- 推送当前分支时，优先显式指定 key，不要依赖默认 SSH identity：

```sh
GIT_SSH_COMMAND='ssh -i ~/.ssh/id_ed25519_auto_money -o IdentitiesOnly=yes -o StrictHostKeyChecking=no' \
git push origin autoresearch/20260328-all-plan
```

## 真实验证规则

### Bitget

这条链路必须围绕同一条真实主线验证：

- `symbol=MSTRUSDT`
- `productType=USDT-FUTURES`
- `marginMode=isolated`
- `leverage=3`
- `direction=long`

所有真实 Bitget 验证都必须满足：

- 先确认或显式清回空仓。
- 真正下单后必须再次清回空仓。
- 不得影响用户其他仓位。
- 不得把 fake server、mock、unit test 冒充成“真实 Bitget 已完成”。

如果 ECS 环境里没有这三个变量，就必须在命令行临时注入，而不是假设远端 shell 已经有：

- `BITGET_API_KEY`
- `BITGET_API_SECRET`
- `BITGET_PASSPHRASE`

已验证有效的 passphrase 是：

- `nomad19980509`

### 最小有效下单量

不要再把 `size=0.01` 当成真实闭环的固定值。

- `minTradeNum=0.01` 只是步长下限，不代表一定满足 `minTradeUSDT=5`。
- 真实下单量必须按“实时 ticker 价格 + 合约规则”动态计算。
- 对 `MSTRUSDT`，在 `2026-03-29` 实测价格约 `125.9 ~ 126.1` 时，满足 `>= 5 USDT` 的最小有效 size 是 `0.04`。
- 因此，任何真实闭环脚本或 runtime 如果还把默认 size 写死成 `0.01`，都应视为 bug，而不是配置偏好。

当前已验证：

- [`scripts/run_mstr_e2e.sh`](./scripts/run_mstr_e2e.sh) 已改为动态计算有效 size。
- 同一脚本会为每次 run 生成独立 `runtime-config.yaml` 与 `runtime-state.db`，避免复用状态库导致的 `event_log.event_id` 冲突。

### OpenAI / Agent

真实 agent 验证也不能依赖“我以为远端有环境变量”。

- 开始前先检查 `OPENAI_API_KEY` presence。
- 当前已验证可用的 real agent 调用方式是：
  - `RUN_OPENAI_REAL=1`
  - `OPENAI_BASE_URL=https://right.codes/codex/v1`
  - model 侧按仓库当前实现走 `gpt-5.4`

## 恢复会话基线

在 `turn_aborted`、`$resume`、用户中断、或切换新会话之后，恢复工作前必须先重新确认：

```sh
git status --short --branch
PATH=/usr/local/go/bin:$PATH go test ./...
PATH=/usr/local/go/bin:$PATH go build ./cmd/lab ./cmd/marketd ./cmd/traderd ./cmd/agentd
for v in BITGET_API_KEY BITGET_API_SECRET BITGET_PASSPHRASE OPENAI_API_KEY; do ...; done
```

如果涉及真实 Bitget 闭环，还要补一次：

```sh
env RUN_BITGET_REAL=1 RUN_BITGET_CLEANUP=1 ... \
PATH=/usr/local/go/bin:$PATH \
go test ./internal/exchange/bitget -run TestRealBitgetEnsureFlatPosition -count=1 -v
```

## Quant Workflow Durable Docs

如果任务通过 repo-local `quant-workflow` 或同类长期 workflow 执行，durable docs 的 canonical root 优先是控制端本机：

- `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/workspace/core.md`
- `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/workspace/quant-platform-功能说明.md`
- `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/workspace/quant-platform-架构文档.md`
- `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/workspace/quant-platform-接口文档.md`
- `/Users/qiukeming/Documents/projects/ob/obsidian/ecs/workspace/quant-platform-通知与机器人操作文档.md`

如果当前环境看不到这个控制端根目录，则使用当前 repo 下的 mirror：

- `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan/workspace/`

单次执行 workspace 固定创建在：

- controller root 或 repo mirror 下的 `workspace/runs/YYYYMMDD_HHMMSS_<title>/`

completion gate：

- 每次 workflow 结束前，上面五份根级 durable docs 都必须更新到终态
- `workspace/runs/...` 下的 `status.md / execution-log.md / 测试记录.md / 测试报告.md / changelog.md / insights.md` 只代表本次执行，不取代根级 durable docs

## Quant Workflow Resume Order

如果通过 `quant-workflow` 恢复上下文，固定顺序必须是：

1. controller `workspace/core.md`；若不存在则 repo mirror `workspace/core.md`
2. controller `AGENTS.md`；若不存在则跳过
3. 当前 repo `AGENTS.md`
4. 当前 run workspace `status.md`
5. 当前 run workspace `execution-log.md`
6. 然后再执行本节已有的远端 baseline 验证

不要只靠当前 shell 历史和残留终端输出猜现场。

## Quant Workflow Testing Gate

如果任务涉及实现、修复或行为变更，testing 固定三层：

1. `proof test`
   - 必须有能证明改动或复现 bug 修复的 unit / integration test
2. `real verification`
   - 必须在远端 ECS 上使用真实 DB、真实 API、真实 secrets 做实际验证
3. `impact safety`
   - 务必不可以对当前真实在跑的仓位、watchlist、promotion、Telegram bot owner、OpenClaw ownership 产生不可控影响

额外规则：

- 真实验证不能拿 mock / fake server 冒充
- 任何会改动全局运行态且不能自动恢复原状的测试，都必须停下请求用户确认
- 触到 Bitget 写路径时，只能沿 `MSTRUSDT` 主线做最小安全闭环，并且前后都要空仓校验

## 提交与产物规则

提交源码前，先区分“代码变更”和“运行产物”。

- 应提交：源码、测试、配置、方案文档、必要脚本。
- 不应提交：远端运行产生的二进制、临时状态库、回放数据、一次性产物目录。

当前仓位里，以下路径应优先视为运行产物，而不是源码：

- `traderd`
- `var/`
- `artifacts/` 下的一次性运行产物

如果 push 失败，不要立刻重做 commit。先检查：

1. `git remote -v`
2. 远端 GitHub key 选择是否正确
3. 是否需要显式 `GIT_SSH_COMMAND`

## 禁止事项

- 禁止把本机目录当成目标代码仓库直接修改。
- 禁止在未登录 ECS 的情况下声称“已经验证”远端行为。
- 禁止把私钥复制到新的长期文件并提交到版本控制。
- 禁止在本目录内创建本地临时代码、临时脚本或构建产物来替代远端操作。
- 禁止把真实交易闭环跑完后留仓离场。
- 禁止把 `go test ./...` 全绿误判成“所有 plan 都已真实完成”。

## 文档更新规则

如果目标 ECS 的登录信息、IP、用户名、主机名、认证方式、远端工作路径、Git 远端、或 GitHub 专用 key 路径发生变化，先更新 [`openclaw 密钥.md`](./openclaw%20密钥.md) 和本文件，再继续后续任务。

Bitget / OpenAI 等敏感凭证不得写入 repo 内文档；真实值只允许保存在远端环境变量或 controller 侧私有 durable memory。
