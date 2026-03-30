package replay

import (
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/market"
	"quantlab/internal/trader"
)

func TestReplayHarnessReusesTraderEngine(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 62000, Stop: 61000, Target: 64000}}
	eng := trader.NewEngine(trader.Config{ArmingState: trader.ArmingArmed, Strategy: strategy})
	harness := NewHarness(eng, NewLongOnlySimulator())
	report, err := harness.Run([]market.MarketEvent{
		market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000},
		market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000060, 0), Close: 62100},
	})
	if err != nil {
		t.Fatalf("run replay: %v", err)
	}
	if report.EventCount != 2 {
		t.Fatalf("unexpected replay report: %+v", report)
	}
	if report.CommandCount != 4 || report.CandidateCount != 2 {
		t.Fatalf("unexpected command counts: %+v", report)
	}
	if report.Simulation.CandidatesApplied != 2 {
		t.Fatalf("unexpected simulation summary: %+v", report.Simulation)
	}
}

func TestReplayHarnessUsesRuntimeReconcilePath(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 62000, Stop: 61000, Target: 64000}}
	eng := trader.NewEngine(trader.Config{ArmingState: trader.ArmingArmed, Strategy: strategy})
	harness := NewHarness(eng, NewLongOnlySimulator())
	report, err := harness.Run([]market.MarketEvent{
		market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000},
		market.PositionEvent{SymbolValue: "BTCUSDT", Ts: time.Unix(1710000001, 0), Qty: 0.02},
	})
	if err != nil {
		t.Fatalf("run replay: %v", err)
	}
	if report.FinalState.ArmingState != trader.ArmingDegraded {
		t.Fatalf("expected degraded final state, got %+v", report.FinalState)
	}
	if report.CommandCount != 3 || report.CandidateCount != 1 {
		t.Fatalf("unexpected replay report after reconcile: %+v", report)
	}
}

type stubStrategy struct {
	signal core.Signal
}

func (strategy *stubStrategy) OnBar(symbol, interval string, bars []core.Bar) core.Signal {
	return strategy.signal
}
