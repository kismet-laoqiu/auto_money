# Historical Sync SOP

## 实例身份

| 项目 | 值 |
| --- | --- |
| 远端仓库 | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| watchlist | `configs/platform/watchlist.yaml` |
| warehouse config | `configs/platform/warehouse.yaml` |
| compose file | `deploy/docker-compose.platform.yml` |
| warehouse skill | `.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh` |
| 高周期目标 | `15m/1h/4h/1d/1w` |
| horizon | `1095` 天 |

这份 SOP 针对当前已经落地的 watchlist + warehouse 方案。目标不是再去维护分散 symbol 列表，而是用单一 watchlist 驱动 PG warehouse 与 live runtime。

## 先决条件

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
docker compose -f deploy/docker-compose.platform.yml ps
PATH=/usr/local/go/bin:$PATH go run ./cmd/platformctl warehouse health -config configs/platform/warehouse.yaml
```

通过标准：

- `warehouse-db` 为 `healthy`
- `warehouse health` 返回 `ok=true`
- `retention_days=1095`

## 标准入口

### 1. 只改 watchlist

新增或删除 symbol 时，只改：

- `configs/platform/watchlist.yaml`

不要再去手工同步：

- `cmd/platformctl/main.go` 的硬编码 symbol 列表
- `configs/live-bitget.yaml` 里的 symbol 数组

当前这些都不再是主入口。

### 2. 构建最新 `platformctl`

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
mkdir -p bin
PATH=/usr/local/go/bin:$PATH go build -o ./bin/platformctl ./cmd/platformctl
```

### 3. 执行统一回补

```bash
./bin/platformctl watchlist apply \
  -watchlist configs/platform/watchlist.yaml \
  -live-config configs/live-bitget.yaml \
  -warehouse-config configs/platform/warehouse.yaml
```

这个命令会：

1. 从 watchlist 读取 `symbols`
2. 对 `15m/1h/4h/1d/1w` 做 horizon backfill
3. 验证 live config 通过 `watchlist_path` 能解析出新的 symbol 集合

## 验证方式

### 1. 看整体 interval coverage

```bash
./.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh interval-coverage
```

### 2. 看单个 symbol 的全部 interval

```bash
./.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh interval-coverage --symbols BTCUSDT --all-intervals
```

### 3. 看 freshness

```bash
./.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh freshness
```

### 4. 看 gap

```bash
./.agents/skills/warehouse-pg-readonly/scripts/warehouse_query.sh gaps --limit 50
```

## 已验证样本

`2026-03-30` 真实验证表明，`BTCUSDT` 已成功写入三年高周期覆盖：

- `15m = 104597`
- `1h = 26149`
- `4h = 6537`
- `1d = 1083`
- `1w = 145`

对应最早时间大约为：

- `15m = 2023-03-31 08:30:00+00`
- `1h = 2023-03-31 09:00:00+00`
- `4h = 2023-03-31 12:00:00+00`
- `1d = 2023-03-31 16:00:00+00`
- `1w = 2023-04-02 16:00:00+00`

## 重要说明

### 1. 当前是长时间 batch job

三年 `15m` 数据量很大。

对完整 watchlist 执行 `watchlist apply` 时，不要把它当成秒级命令。它更像一次运维批处理作业。

### 2. 当前 PG 不是实时仓库

当前 PG `market_bars` 只由 historical / aggregate 链路更新。

live runtime 事件仍然写 sqlite `state_db`，不是持续写入 PG。

### 3. 当前 live 不是 hot reload

watchlist 改完并执行 `watchlist apply` 之后：

- 配置层会自动识别新 symbol
- 正在运行的 `marketd` 仍然需要重启，才能真正开始订阅新 symbol

### 4. Bitget `history-candles` 的真实限制

`2026-03-30` 已实测确认：

- 同时传 `startTime + endTime` 时，Bitget `history-candles` 最大窗口只有 `90` 天
- 当前代码已经改成：长窗口只传 `endTime + limit` 反向翻页

因此，如果后续再看到 `40017 Parameter verification failed startTime || endTime`，优先检查是否有人把长窗口 `startTime` 又加回去了。
