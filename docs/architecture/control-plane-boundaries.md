# Control Plane Boundaries

## platformd

- serves operator HTTP API
- exposes status, positions, orders, events, bars, features, backtest, promotion, live flatten
- reads SQLite runtime state and strategy registry
- may request live actions only through `live.Service`, which shells into `execd`

## mcpd

- exposes the same control plane over stdio MCP
- supports `initialize`, `tools/list`, and `tools/call`
- read tools are always safe
- write tools require `auth_token`
- MCP never receives exchange credentials directly

## OpenClaw

- acts as conversational gateway and delivery plane
- reads platform state via `platformd` HTTP API or `platformctl`
- delivers results back to Telegram through OpenClaw channels
- does not hold Bitget write credentials
- must not become the exchange truth source

## execd

- is the only Bitget write path
- owns order placement and flatten execution
- remains outside OpenClaw and MCP trust boundary

## operator rule

- use `platformd` for HTTP operators
- use `mcpd` for AI tool calls
- use OpenClaw for conversational ingress and Telegram delivery
- never bypass `execd` with direct exchange writes from AI surfaces
