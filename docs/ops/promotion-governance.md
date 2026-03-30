# Promotion Governance SOP

## 实例概览

| 项目 | 值 |
| --- | --- |
| 平台进程地址 | `http://127.0.0.1:18087` |
| 示例配置文件 | `configs/demo-mstr-bundle.yaml` |
| 示例策略 | `mstr-wave-fib` |
| 示例版本 | `v0.1.0` |
| approve 样例 promotion_id | `mstr-wave-fib-v0-1-0-1774782602750969947` |
| rollback 样例 promotion_id | `mstr-wave-fib-v0-1-0-1774782602853963264` |
| backtest JSON 产物 | `artifacts/mstr-bundle/backtest-result.json` |
| backtest Markdown 产物 | `artifacts/mstr-bundle/backtest-report.md` |
| 临时 state DB | `/tmp/p08-promotion.db` |

本实例的真实结果如下：

- `mstr-wave-fib-v0-1-0-1774782602750969947` 最终状态是 `live_active`。
- `mstr-wave-fib-v0-1-0-1774782602853963264` 最终状态是 `rolled_back`。
- `artifacts/mstr-bundle/backtest-result.json` 记录 `metric_name=final_score`、`symbol=MSTRUSDT`、`bars=720`、`final_score=-109.52527002520578`。
- `artifacts/mstr-bundle/backtest-report.md` 记录 aggregate metrics 和单 dataset 摘要，适合 operator 快速阅读。

## 操作主线

### 第一步：创建 promotion request

执行命令：

```bash
./bin/platformctl promotion request \
  -addr http://127.0.0.1:18087 \
  -strategy mstr-wave-fib \
  -version v0.1.0 \
  -config configs/demo-mstr-bundle.yaml
```

从这一步可以得知：

- `POST /api/promotions/request` 会同步触发一次 backtest。
- 返回体里的 `state` 固定先落到 `backtest_passed`。
- 返回体里的 `summary` 直接暴露本次 backtest 的 `objective_score` 与 `final_score`。
- `config_path` 会写入 promotion details，后续回查不需要猜配置来源。

### 第二步：跑 approve 主线

执行命令：

```bash
./bin/platformctl promotion start-shadow -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602750969947
./bin/platformctl promotion pass-shadow  -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602750969947
./bin/platformctl promotion start-canary -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602750969947
./bin/platformctl promotion approve      -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602750969947
```

从这一步可以得知：

- `start-shadow` 只接受 `backtest_passed`。
- `pass-shadow` 只接受 `shadow_running`。
- `start-canary` 只接受 `shadow_passed`。
- `approve` 只接受 `canary_running`，成功后状态进入 `live_active`。

approve 样例最终返回：

| 字段 | 值 |
| --- | --- |
| `state` | `live_active` |
| `title` | `mstr-wave-fib v0.1.0 promotion approved` |
| `summary` | `promotion is live_active` |
| `details` | `promotion_id=...`、`strategy=mstr-wave-fib`、`version=v0.1.0`、`config=configs/demo-mstr-bundle.yaml` |

### 第三步：跑 rollback 主线

执行命令：

```bash
./bin/platformctl promotion start-shadow -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602853963264
./bin/platformctl promotion pass-shadow  -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602853963264
./bin/platformctl promotion start-canary -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602853963264
./bin/platformctl promotion rollback     -addr http://127.0.0.1:18087 -id mstr-wave-fib-v0-1-0-1774782602853963264 -reason 'operator smoke rollback'
```

从这一步可以得知：

- `rollback` 接受 `canary_running`、`canary_degraded`、`live_active` 三种来源状态。
- `rollback` 的 `reason` 会写入 `details`，便于 notifier 和 operator 回查。
- rollback 成功后状态固定写成 `rolled_back`。
- `promotion list` 会按 `updated_at` 倒序返回最新状态。

rollback 样例最终返回：

| 字段 | 值 |
| --- | --- |
| `state` | `rolled_back` |
| `title` | `mstr-wave-fib v0.1.0 rolled back` |
| `summary` | `promotion rolled back` |
| `details` | 额外包含 `reason=operator smoke rollback` |

## 状态速查表

| 当前状态 | 动作 | API | CLI | 下一个状态 |
| --- | --- | --- | --- | --- |
| `backtest_passed` | `start_shadow` | `POST /api/promotions/shadow` | `platformctl promotion start-shadow` | `shadow_running` |
| `shadow_running` | `pass_shadow` | `POST /api/promotions/shadow/pass` | `platformctl promotion pass-shadow` | `shadow_passed` |
| `shadow_passed` | `start_canary` | `POST /api/promotions/canary` | `platformctl promotion start-canary` | `canary_running` |
| `canary_running` | `approve` | `POST /api/promotions/approve` | `platformctl promotion approve` | `live_active` |
| `canary_running` | `degrade_canary` | `POST /api/promotions/canary/degrade` | `platformctl promotion degrade-canary` | `canary_degraded` |
| `canary_running` | `rollback` | `POST /api/promotions/rollback` | `platformctl promotion rollback` | `rolled_back` |
| `canary_degraded` | `rollback` | `POST /api/promotions/rollback` | `platformctl promotion rollback` | `rolled_back` |
| `live_active` | `rollback` | `POST /api/promotions/rollback` | `platformctl promotion rollback` | `rolled_back` |

## 产物与回查

`backtest-result.json` 与 `backtest-report.md` 是 promotion request 的前置证据。operator 在 approve 之前至少检查以下项目：

- `metric_name` 必须是 `final_score`。
- `reports[0].symbol` 必须与策略 universe 一致，本例是 `MSTRUSDT`。
- `reports[0].bars` 必须覆盖本次回测窗口，本例是 `720`。
- `aggregate.dataset_count` 必须和本次 dataset 数量一致，本例是 `1`。

建议回查顺序如下：

1. 打开 `artifacts/mstr-bundle/backtest-report.md`，先看 aggregate metrics 和 `signal_count`。
2. 打开 `artifacts/mstr-bundle/backtest-result.json`，再看 `reports[0].trades` 和 `robust.bucket_means`。
3. 执行 `./bin/platformctl promotion get -addr http://127.0.0.1:18087 -id <promotion_id>`，确认平台当前状态与 backtest 证据一致。
4. 执行 `./bin/platformctl promotion list -addr http://127.0.0.1:18087`，确认没有并发 promotion 处于冲突状态。

## 边界条件

- `approve` 不能跳过 `shadow` 和 `canary`。
- `degrade-canary` 不能从 `backtest_passed` 或 `shadow_passed` 直接执行。
- `rollback` 必须带 `-reason`，否则 operator 无法从 `details` 复原现场。
- `promotion request` 当前会复用 `backtest.RunConfig`，因此 backtest 失败会直接阻断 promotion 创建。

## 代码入口

| 入口 | 文件 |
| --- | --- |
| promotion state machine | `internal/platform/promotion/state_machine.go` |
| shadow helper | `internal/platform/promotion/shadow.go` |
| canary helper | `internal/platform/promotion/canary.go` |
| rollback helper | `internal/platform/promotion/rollback.go` |
| platform API router | `internal/platform/api/router.go` |
| CLI command | `cmd/platformctl/main.go` |
