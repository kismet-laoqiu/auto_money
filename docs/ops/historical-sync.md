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
- `configs/live.yaml` 里的 symbol 数组

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
  -live-config configs/live.yaml \
  -warehouse-config configs/platform/warehouse.yaml
```

这个命令会：

1. 从 watchlist 读取 `symbols`
2. 对 `15m/1h/4h/1d/1w` 做 horizon backfill
3. 验证 live config 通过 `watchlist_path` 能解析出新的 symbol 集合

### 4. 构建并重启 `marketd`

```bash
PATH=/usr/local/go/bin:$PATH go build -o ./bin/marketd ./cmd/marketd
sudo cp deploy/systemd/quantlab-marketd.service /etc/systemd/system/quantlab-marketd.service
sudo systemctl daemon-reload
sudo systemctl restart quantlab-marketd
systemctl status quantlab-marketd --no-pager -l
```

通过标准：

- `ExecStart` 指向 `configs/live.yaml`
- `marketd` 进程正常运行
- 新增 symbol 已经被订阅

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

同日已补齐 live warehouse 链路：

- `marketd` 读取 `configs/live.yaml`
- `configs/live.yaml` 通过 `watchlist_path` 解析 15 个 symbol
- `marketd` 通过 `warehouse_config_path` 连接 PG warehouse
- 每个 watchlist symbol 订阅 `15m/1h/4h/1d/1w` 五档 candle channel
- PG `market_bars` 只写 closed bar，不写未收盘的进行中 bar

## 重要说明

### 1. 当前是长时间 batch job

三年 `15m` 数据量很大。

对完整 watchlist 执行 `watchlist apply` 时，不要把它当成秒级命令。它更像一次运维批处理作业。

### 2. 当前 PG 是 closed-bar 实时仓库

当前 PG `market_bars` 分成两条写入路径：

- historical / aggregate：负责历史回补与离线补齐
- `marketd` live runtime：负责 `15m/1h/4h/1d/1w` closed bar 持续写入

当前 sqlite `state_db` 仍然保留为 live runtime 事件流真相源，给 `traderd`、cursor、checkpoint 使用。

这两个层次不冲突：

- PG 负责历史研究、回测、导出、仓库分析
- sqlite 负责运行态 event log

要注意，PG 实时写入的是 closed bar：

- `15m/1h/4h/1d/1w` 只会在 bar 真正收盘后推进
- `1d/1w` 的 freshness 天然慢于分钟级，不要把这个误判成异常

### 3. 当前 live 不是 hot reload

watchlist 改完并执行 `watchlist apply` 之后：

- 配置层会自动识别新 symbol
- 正在运行的 `marketd` 仍然需要重启，才能真正开始订阅新 symbol 并把 closed bar 持续写入 PG

### 4. Bitget `history-candles` 的真实限制

`2026-03-30` 已实测确认：

- 同时传 `startTime + endTime` 时，Bitget `history-candles` 最大窗口只有 `90` 天
- 当前代码已经改成：长窗口只传 `endTime + limit` 反向翻页

因此，如果后续再看到 `40017 Parameter verification failed startTime || endTime`，优先检查是否有人把长窗口 `startTime` 又加回去了。
