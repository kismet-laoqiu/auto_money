# Historical Sync SOP

## 实例身份

| 项目 | 值 |
| --- | --- |
| 远端仓库 | `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan` |
| warehouse config | `configs/platform/warehouse.yaml` |
| compose file | `deploy/docker-compose.platform.yml` |
| artifacts 根目录 | `artifacts/platform/historical-sync` |
| 缺省 symbols | `BTCUSDT ETHUSDT SOLUSDT MSTRUSDT CRCLUSDT HOODUSDT BABAUSDT TAOUSDT EWYUSDT AAPLUSDT LINKUSDT SEIUSDT WLDUSDT CLUSDT DOGEUSDT` |
| 固定基础粒度 | `1m/5m/15m/1h/4h/1d` |
| 已验证全量回补 artifact | `artifacts/platform/historical-sync/20260329T120611Z` |
| 已验证幂等重跑 artifact | `artifacts/platform/historical-sync/20260329T120625Z` |

本文围绕 Bitget `USDT-FUTURES` 历史回补这条真实主线编写，目标是把数据稳定写入 Timescale warehouse，而不是只拉一遍临时 JSON。

## 第一步：确认 warehouse 已可写

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
docker compose -f deploy/docker-compose.platform.yml ps
PATH=/usr/local/go/bin:$PATH ./bin/platformctl warehouse health   -config configs/platform/warehouse.yaml
```

如果 `warehouse-db` 不是 `healthy`，先回到 `docs/ops/warehouse-setup.md` 完成 compose 与 migration。

## 第二步：构建最新 `platformctl`

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
PATH=/usr/local/go/bin:$PATH go build -o ./bin/platformctl ./cmd/platformctl
```

这样可以确保 `historical sync` 使用当前代码里的缺省 allowlist 和最新 artifact metadata。

## 第三步：执行首轮回补

如果不传 `-symbols`，命令会自动使用 `live_symbol_allowlist` 对应的 15 个 symbol：

```bash
cd /root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan
./bin/platformctl historical sync   -warehouse-config configs/platform/warehouse.yaml   -provider bitget   -product-type USDT-FUTURES   -intervals 1h   -limit 20
```

已验证结果：

- `generated_at=2026-03-29T12:06:11Z`
- `artifact_dir=artifacts/platform/historical-sync/20260329T120611Z`
- 15 个 symbol 都写入成功
- `gap_count=0`
- 大多数 symbol `inserted=20 row_count=20`
- `MSTRUSDT` 因此前已存在一轮 `1h` 数据，结果是 `inserted=1 row_count=21`

## 第四步：验证写库结果

```bash
docker compose -f deploy/docker-compose.platform.yml exec -T warehouse-db   psql -U quantlab -d quantlab   -c "select symbol, interval, count(*) as row_count from market_bars where provider='bitget' and symbol in ('BTCUSDT','ETHUSDT','SOLUSDT','MSTRUSDT','CRCLUSDT','HOODUSDT','BABAUSDT','TAOUSDT','EWYUSDT','AAPLUSDT','LINKUSDT','SEIUSDT','WLDUSDT','CLUSDT','DOGEUSDT') and interval='1h' group by symbol, interval order by symbol;"
```

`2026-03-29` 的真实查询结果是 15 行都存在，其中 `MSTRUSDT=21`，其余 `=20`。

## 第五步：验证幂等重跑

在同一窗口再次执行完全相同的命令：

```bash
./bin/platformctl historical sync   -warehouse-config configs/platform/warehouse.yaml   -provider bitget   -product-type USDT-FUTURES   -intervals 1h   -limit 20
```

已验证结果：

- `generated_at=2026-03-29T12:06:25Z`
- `artifact_dir=artifacts/platform/historical-sync/20260329T120625Z`
- 15 个 symbol 全部 `inserted=0`
- `row_count` 保持稳定，没有异常增长

这说明同一窗口重跑不会重复写入，是当前的失败恢复基线。

## 第六步：查看 artifact 报表

每次运行都会生成：

- `summary.json`：包含 `provider/product_type/symbols/intervals/limit` 请求元数据，以及每个 dataset 的 `inserted/row_count/checksum/gap_count`
- `gap-report.json`：只保留 `symbol/interval/gap_count`，方便 operator 快速看补洞结果

示例：

```bash
jq '.' artifacts/platform/historical-sync/20260329T120611Z/summary.json
jq '.' artifacts/platform/historical-sync/20260329T120611Z/gap-report.json
```

## 时间窗口与速率约束

当前实现按 `limit` 控制单次请求 bars 数量，没有额外的 client-side sleep。

- 小窗口验证：`-limit 20`
- 常规首轮回补：先用 `1h` 或 `4h` 验证 symbol 覆盖，再按需要扩到 `1m/5m/15m/1d`
- 不要一上来对 15 个 symbol 同时打满 6 档大窗口；先跑一档确认链路健康，再扩量

## 失败恢复

如果某轮执行失败，不需要清库，直接按最小范围重跑：

- 单 symbol：`-symbols MSTRUSDT`
- 多 symbol：`-symbols BTCUSDT,ETHUSDT,SOLUSDT`
- 单 interval：`-intervals 1h`
- 多 interval：`-intervals 1m,5m,15m,1h,4h,1d`

恢复原则：

- 先缩小到失败的 `symbol + interval` 组合
- 先确认上一次 artifact 里 `gap_count` 和 `row_count` 是什么
- 重跑后以 `inserted` 是否归零、`row_count` 是否稳定作为恢复成功标准
- 不要先删表再重灌；当前写路径已经是 `ON CONFLICT DO NOTHING`
