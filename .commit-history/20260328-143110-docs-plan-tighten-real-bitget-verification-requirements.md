# docs(plan): 收紧真实 Bitget 验证门槛

## 问题描述
1. 当前 runtime closure plan 虽然要求接入 Bitget futures，但完成条件主要停留在本地测试和 smoke command，因为没有把真实 Bitget 公共和私有读链路、下单查询、订单生命周期验证写成硬门槛，导致实现阶段可能在 mock 通过后就错误宣称完成。
2. 私有 REST、private websocket 和下单链路依赖 passphrase 签名，因为 plan 没有把 `BITGET_PASSPHRASE` 写成硬前置条件，导致账户查询、仓位同步和真实下单验证存在被模糊跳过的风险。
3. 凭证注入方式如果不明确，因为实现人员可能把 key 和 secret 写进 YAML 或文档，导致 ECS 上的真实验证流程和 Git 安全边界同时失真。

## 解决方案
1. 在 plan 顶部新增 Real Bitget Verification Contract，把 public REST、public websocket、private REST、private websocket 和真实 order lifecycle 全部升级为完成门槛，落点为 plan 文档头部与 Task 2、Task 3、Task 5、Task 8。
2. 在任务步骤和 Verification Matrix 中显式加入 `RUN_BITGET_REAL=1`、真实 ECS 凭证映射以及 order query 和 reduce-only exit 验证，直接把缺失 passphrase 的情况定义为任务未完成，落点为 Task 2、Task 3、Task 5、Task 8 和 Verification Matrix。
3. 把凭证处理收敛到 env-only contract，明确 `BITGET_API_KEY`、`BITGET_API_SECRET`、`BITGET_PASSPHRASE` 只允许通过环境变量注入，不允许写入 tracked 文件，落点为 Real Bitget Verification Contract、Assumptions、Plan Self-Review。

## 修改文件
- `plan/2026-03-28-agentic-runtime-closure-plan.md`: 新增真实 Bitget 验证合同，并把私有链路与真实 order lifecycle 写成强制验收项。

## 影响范围
- [API] 无
- [配置] 文档新增 ECS 环境变量契约：`BITGET_API_KEY`、`BITGET_API_SECRET`、`BITGET_PASSPHRASE`
- [DB] 无
- [破坏性] 无

## 测试验证
- 远端 `sed -n '1,80p' plan/2026-03-28-agentic-runtime-closure-plan.md`
- 远端 `rg -n 'Real Bitget Verification Contract|BITGET_PASSPHRASE|RUN_BITGET_REAL|order lifecycle|mock pass != done' plan/2026-03-28-agentic-runtime-closure-plan.md`
- 远端 `rg -n 'TODO|TBD|implement later|fill in details|placeholder' plan/2026-03-28-agentic-runtime-closure-plan.md`
- 远端 `git status --short --branch`
