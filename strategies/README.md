# Strategy Bundle Layout

## 实例身份

| 项目 | 值 |
| --- | --- |
| `strategy_id` | `mstr-wave-fib` |
| `version` | `v0.1.0` |
| bundle 根目录 | `strategies/mstr-wave-fib/versions/v0.1.0` |
| bundle-backed config | `configs/demo-mstr-bundle.yaml` |
| registry API | `GET /api/strategies/versions?strategy_id=mstr-wave-fib` |

当前仓库里唯一已注册的 bundle 目录树是：

```text
strategies/
└── mstr-wave-fib/
    └── versions/
        └── v0.1.0/
            ├── gates.cel
            ├── risk.yaml
            ├── score.cel
            ├── strategy.yaml
            └── universe.yaml
```

## 第一步：先看 `strategy.yaml`

`strategies/mstr-wave-fib/versions/v0.1.0/strategy.yaml` 当前定义了：

- `strategy_id: mstr-wave-fib`
- `version: v0.1.0`
- `description: Baseline MSTR wave and fib strategy bundle mirroring the current hardcoded runtime.`
- `strategy.fast_sma: 15`
- `strategy.slow_sma: 60`
- `strategy.signal_threshold: 3.25`
- `objective.drawdown_weight: 2.0`
- `objective.pnl_weight: 0.2`

从这个文件可以得知：

- bundle 身份由 `strategy_id + version` 唯一确定。
- 当前 `mstr-wave-fib` 的信号参数已经脱离硬编码，进入版本化文件。
- 当前 objective 也随 bundle 版本一起走，不再要求 operator 在临时 config 里重复维护。

## 第二步：再看 `universe.yaml`

`strategies/mstr-wave-fib/versions/v0.1.0/universe.yaml` 当前定义了：

- dataset 名称：`mstrusdt_demo_replay`
- provider：`bitget`
- symbol：`MSTRUSDT`
- interval：`1m`
- limit：`720`
- product_type：`USDT-FUTURES`
- symbols：`[MSTRUSDT]`

从这个文件可以得知：

- 回测与查询链路读取的数据集入口已经固定到 bundle。
- `configs/demo-mstr-bundle.yaml` 不需要再手写 `datasets`，`strategybundle.LoadConfig` 会把这里的 `datasets` 覆盖进运行时配置。
- 当前策略的研究主线是 `MSTRUSDT / 1m / 720 bars`。

## 第三步：看 `risk.yaml`

`strategies/mstr-wave-fib/versions/v0.1.0/risk.yaml` 当前定义了：

- `max_leverage: 3`
- `margin_mode: isolated`
- `product_type: USDT-FUTURES`
- `symbols[0].symbol: MSTRUSDT`
- `symbols[0].max_notional: 30`
- `symbols[0].max_tranches: 1`

从这个文件可以得知：

- live 路径的 leverage、margin mode、product type、symbol allowlist 已经进入 bundle。
- `marketd`、`traderd`、`execd`、`platformd`、`agentd`、`stream` 现在都通过 `strategybundle.LoadConfig` 读取这些覆盖字段。
- `configs/demo-mstr-bundle.yaml` 不再重复写 `product_type`、`margin_mode`、`symbols`、`max_leverage`。

## 第四步：看 `score.cel` 与 `gates.cel`

当前文件是：

- `score.cel` -> `final_score`
- `gates.cel` -> 含 `shadow` gate 程序

从这两个文件可以得知：

- bundle 不只承载参数，也承载评分和上线 gate 的表达式入口。
- 当前 registry 把版本目录视作完整 bundle；缺任意一个文件，`Validate` 都会直接拒绝。

## 第五步：看 bundle-backed config

`configs/demo-mstr-bundle.yaml` 当前只保留：

- `cache_dir`
- `artifact_dir`
- `strategy_bundle_path: ../strategies/mstr-wave-fib/versions/v0.1.0`
- `live.runtime.*`
- `live.exchange` 的连接信息与凭据环境变量名
- `live.agent.*`

从这个文件可以得知：

- strategy/objective/datasets/live risk 四组字段已经可以从 bundle 单点注入。
- `strategybundle.LoadConfig` 会先按 config 文件目录解析相对 `strategy_bundle_path`，再把 bundle 覆盖回 `config.Config`。
- 只要 config 指向新的 bundle 版本，回测、query、market、trader、exec、platform 就会一起看到同一套参数。

## 第六步：看 registry 出口

当前 registry 是文件系统存储层，扫描规则是：

```text
strategies/<strategy_id>/versions/<version>/
```

`platformd` 当前会把 `config` 同目录旁边的 `../strategies` 作为 registry 根目录。真实 smoke 返回：

```json
[
  {
    "strategy_id": "mstr-wave-fib",
    "version": "v0.1.0",
    "root_path": "strategies/mstr-wave-fib/versions/v0.1.0",
    "config_path": "",
    "updated_at": "2026-03-29T10:22:28Z"
  }
]
```

从这个出口可以得知：

- registry 已经能 list 真实 bundle 版本，而不是只靠 promotion 历史倒推。
- `root_path` 已成为 operator 可读的单一版本定位信息。
- promotion 状态现在是 enrich 层，不再承担“版本发现”职责。

## 速查

| 文件 | 作用 | 当前真实来源 |
| --- | --- | --- |
| `strategy.yaml` | 策略身份 + 参数 + objective | `strategies/mstr-wave-fib/versions/v0.1.0/strategy.yaml` |
| `universe.yaml` | dataset + symbol universe | `strategies/mstr-wave-fib/versions/v0.1.0/universe.yaml` |
| `risk.yaml` | live risk + symbol allowlist | `strategies/mstr-wave-fib/versions/v0.1.0/risk.yaml` |
| `score.cel` | score 入口 | `strategies/mstr-wave-fib/versions/v0.1.0/score.cel` |
| `gates.cel` | promotion gate 入口 | `strategies/mstr-wave-fib/versions/v0.1.0/gates.cel` |
| `demo-mstr-bundle.yaml` | bundle-backed runtime config | `configs/demo-mstr-bundle.yaml` |

## 参考入口

- `internal/strategybundle/loader.go`
- `internal/strategybundle/validator.go`
- `internal/strategybundle/resolve.go`
- `internal/strategybundle/registry.go`
- `cmd/platformd/main.go`
- `internal/platform/api/router.go`
