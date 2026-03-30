# Strategy Bundle Authoring SOP

## 实例身份

| 项目 | 值 |
| --- | --- |
| 当前参考 bundle | `strategies/mstr-wave-fib/versions/v0.1.0` |
| 当前参考 config | `configs/demo-mstr-bundle.yaml` |
| 当前 registry smoke 端口 | `127.0.0.1:18084` |
| 当前 backtest smoke 端口 | `127.0.0.1:18085` |

本文的所有操作都围绕 `mstr-wave-fib / v0.1.0` 这条真实主线展开。

## 第一步：复制现有版本目录

以从 `v0.1.0` 派生 `v0.1.1` 为例：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
cp -R strategies/mstr-wave-fib/versions/v0.1.0 \
  strategies/mstr-wave-fib/versions/v0.1.1
```

从这个操作可以得知：

- 版本目录是最小发布单元。
- 新版本必须在同一 `strategy_id` 下新增目录，不能覆盖旧版本目录。

## 第二步：修改 `strategy.yaml`

至少修改以下字段：

- `version`
- `description`
- `strategy.*`
- `objective.*`

建议先改这两个身份字段：

```yaml
strategy_id: mstr-wave-fib
version: v0.1.1
description: MSTR wave and fib bundle with adjusted entry threshold.
```

从这个文件可以得知：

- `strategy_id` 不变表示同一策略族。
- `version` 变化表示新候选版本。
- 如果参数与 description 不一致，registry 虽然能发现版本，但 operator 很难区分版本目的。

## 第三步：修改 `universe.yaml` 与 `risk.yaml`

必须核对三类字段：

| 文件 | 必查字段 | 当前 `v0.1.0` 真实值 |
| --- | --- | --- |
| `universe.yaml` | `datasets[*].symbol` / `interval` / `limit` | `MSTRUSDT / 1m / 720` |
| `risk.yaml` | `max_leverage` | `3` |
| `risk.yaml` | `margin_mode` / `product_type` | `isolated / USDT-FUTURES` |
| `risk.yaml` | `symbols[*].max_notional` / `max_tranches` | `30 / 1` |

从这里可以得知：

- 回测数据集和 live risk 必须一起变更审阅，不能只改 strategy 参数。
- bundle-backed config 不再覆盖这些字段，operator 看到的就是这里的真实值。

## 第四步：修改 `score.cel` 与 `gates.cel`

当前 `v0.1.0` 的 smoke 已证明：

- `score.cel` 可被 loader 读到
- `gates.cel` 含 `shadow` gate，可被 validator 接受

修改后必须保证两个文件都非空，并且仍然是纯文本文件。

## 第五步：指向新 bundle 版本

如果要让 backtest 和 platform smoke 读取新版本，把 config 改到目标版本：

```yaml
strategy_bundle_path: ../strategies/mstr-wave-fib/versions/v0.1.1
```

当前仓库已经提供最小 bundle-backed config：

- `configs/demo-mstr-bundle.yaml`

从这里可以得知：

- strategy/objective/datasets/live risk 的唯一入口是 `strategy_bundle_path` 指向的目录。
- `strategybundle.LoadConfig` 会相对 config 文件目录解析路径，不依赖当前 shell 的 `cwd`。

## 第六步：做包级校验

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
PATH=/usr/local/go/bin:$PATH go test ./internal/strategybundle -count=1 -v
```

从这个校验可以得知：

- loader、validator、registry、relative path resolve 全部仍然通过。
- 新版本目录如果缺文件或 YAML 字段非法，会在 registry/list 或 bundle/load 阶段失败。

## 第七步：做 registry smoke

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
PATH=/usr/local/go/bin:$PATH go build -o ./bin/platformd ./cmd/platformd
PATH=/usr/local/go/bin:$PATH go build -o ./bin/platformctl ./cmd/platformctl
./bin/platformd -config configs/demo-mstr-bundle.yaml -listen 127.0.0.1:18084
```

另一个 shell 执行：

```bash
curl -fsS 'http://127.0.0.1:18084/api/strategies/versions?strategy_id=mstr-wave-fib' | jq '.'
./bin/platformctl strategy versions -addr http://127.0.0.1:18084 -strategy mstr-wave-fib
```

从这个 smoke 可以得知：

- 新版本已经被 registry 发现。
- `root_path` 已经出现在 operator API 和 CLI 中。
- promotion 为空并不妨碍版本被枚举。

## 第八步：做 backtest smoke

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
./bin/platformd -config configs/demo-mstr-bundle.yaml -listen 127.0.0.1:18085
./bin/platformctl backtest run -addr http://127.0.0.1:18085 -config configs/demo-mstr-bundle.yaml
```

`2026-03-29` 的真实返回已经证明：

- bundle-backed config 能跑出 `MSTRUSDT / 720 bars` 报告
- `final_score = -109.52527002520578`
- `objective_score = -136.90658753150723`

从这个结果可以得知：

- 新 bundle 不只是能被发现，还能真实进入 deterministic backtest 主链。
- 如果这里失败，优先回看 `strategy_bundle_path`、`universe.yaml`、`risk.yaml`。

## 发布前检查表

| 检查项 | 命令 | 成功标准 |
| --- | --- | --- |
| bundle 包测试 | `go test ./internal/strategybundle -count=1 -v` | 全绿 |
| registry API | `curl /api/strategies/versions?...` | 返回新 `version` 与 `root_path` |
| operator CLI | `platformctl strategy versions ...` | 返回新 `version` |
| backtest smoke | `platformctl backtest run ...` | 返回报告 JSON |
| bundle-backed config | `configs/demo-mstr-bundle.yaml` | 只保留连接信息与 `strategy_bundle_path` |
