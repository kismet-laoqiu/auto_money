# Theory Index

## Purpose

This index ties the platform's theory reading path to the feature groups that are actually materialized, snapshotted, and tested.

## Read Order

1. `docs/zh/system-design.md`
2. `docs/zh/elliott-wave.md`
3. `docs/zh/support-resistance.md`
4. `docs/zh/fibonacci.md`
5. `docs/zh/price-action.md`

## Feature Groups

- `wave_structure`
  - feature spec: `docs/features/wave-structure.md`
  - snapshot test: `internal/core/feature_snapshots_test.go` -> `TestWaveStructureSnapshots`
- `level_cluster`
  - feature spec: `docs/features/level-cluster.md`
  - snapshot test: `internal/core/feature_snapshots_test.go` -> `TestLevelClusterSnapshots`
- `fib_confluence`
  - feature spec: `docs/features/fib-confluence.md`
  - snapshot test: `internal/core/feature_snapshots_test.go` -> `TestFibConfluenceSnapshots`
- `price_action_trigger`
  - feature spec: `docs/features/price-action-trigger.md`
  - snapshot test: `internal/core/feature_snapshots_test.go` -> `TestPriceActionTriggerSnapshots`
- `volume_confirmation`
  - feature spec: `docs/features/volume-confirmation.md`
  - snapshot test: `internal/core/feature_snapshots_test.go` -> `TestVolumeConfirmationSnapshots`
- `regime_tags`
  - feature spec: `docs/features/regime-tags.md`
  - snapshot test: `internal/core/feature_snapshots_test.go` -> `TestRegimeSnapshots`

## Warehouse Materialization

- feature version contract: `internal/warehouse/features/version.go`
- materialize + fetch API: `internal/warehouse/features/materializer.go`
- Postgres persistence: `internal/warehouse/features/store.go`
- real warehouse round-trip verification: `internal/warehouse/features/materializer_integration_test.go`

## Operator Note

The warehouse snapshot source of truth is `strategy_feature_snapshots` in PostgreSQL. The platform keeps the feature formulas in `internal/core`, then uses `internal/warehouse/features` to materialize the same feature set into warehouse rows for later comparison, replay, and promotion evidence.
