package trader

import (
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
)

func TestEngineTurnsClosedBarIntoCandidate(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 62000, Stop: 61000, Target: 64000, Reasons: []string{"legacy"}}}
	engine := NewEngine(Config{ArmingState: ArmingSafe, Strategy: strategy})
	cmds, err := engine.Advance(market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Open: 61900, High: 62100, Low: 61850, Close: 62000, Volume: 2})
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected one command, got %d", len(cmds))
	}
	candidate, ok := cmds[0].(Candidate)
	if !ok {
		t.Fatalf("unexpected command type %T", cmds[0])
	}
	if candidate.Symbol != "BTCUSDT" || candidate.Interval != "1m" {
		t.Fatalf("unexpected candidate identity: %+v", candidate)
	}
	if candidate.Score != 4.2 || candidate.Side != core.Long {
		t.Fatalf("unexpected candidate signal: %+v", candidate)
	}
	if strategy.calls != 1 || len(strategy.bars) != 1 {
		t.Fatalf("strategy did not observe the closed bar: calls=%d bars=%d", strategy.calls, len(strategy.bars))
	}
}

func TestLegacyRuleProfileDelegatesToSignalFn(t *testing.T) {
	called := false
	profile := LegacyRuleProfile{
		StrategyCfg: config.StrategyConfig{FastSMA: 3},
		Evaluate: func(bars []core.Bar, idx int, cfg config.StrategyConfig) core.Signal {
			called = true
			if len(bars) != 2 || idx != 1 {
				t.Fatalf("unexpected bars passed to signal fn: len=%d idx=%d", len(bars), idx)
			}
			if cfg.FastSMA != 3 {
				t.Fatalf("unexpected strategy cfg: %+v", cfg)
			}
			return core.Signal{Side: core.Long, Score: 3.3, Entry: 101, Stop: 99, Target: 104, Reasons: []string{"delegated"}}
		},
	}
	signal := profile.OnBar("BTCUSDT", "1m", []core.Bar{{Close: 100}, {Close: 101}})
	if !called {
		t.Fatalf("expected custom signal fn to be called")
	}
	if signal.Side != core.Long || signal.Score != 3.3 {
		t.Fatalf("unexpected signal: %+v", signal)
	}
}

func TestTraderSafeModeDoesNotPlaceOrders(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 62000, Stop: 61000, Target: 64000}}
	exchange := &fakeExchange{}
	engine := NewEngine(Config{ArmingState: ArmingSafe, Strategy: strategy, Exchange: exchange})
	_, err := engine.Advance(market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000})
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if exchange.PlaceCalls != 0 {
		t.Fatalf("safe mode must not place orders")
	}
}

func TestTraderArmedModePlacesOrders(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 62000, Stop: 61000, Target: 64000}}
	exchange := &fakeExchange{}
	engine := NewEngine(Config{ArmingState: ArmingArmed, Strategy: strategy, Exchange: exchange})
	_, err := engine.Advance(market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0), Close: 62000})
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if exchange.PlaceCalls != 1 {
		t.Fatalf("expected one place call, got %d", exchange.PlaceCalls)
	}
	if exchange.LastReq.Symbol != "BTCUSDT" || exchange.LastReq.Side != "buy" {
		t.Fatalf("unexpected order request: %+v", exchange.LastReq)
	}
}

func TestEngineTurnsMicroBarIntoCandidate(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Long, Score: 4.2, Entry: 62000, Stop: 61000, Target: 64000}}
	engine := NewEngine(Config{ArmingState: ArmingSafe, Strategy: strategy})
	cmds, err := engine.Advance(market.MicroBarClosedEvent{SymbolValue: "BTCUSDT", OpenedAt: time.Unix(1710000000, 0), ClosedAt: time.Unix(1710000001, 0), Open: 61990, High: 62010, Low: 61980, Close: 62000, Volume: 1})
	if err != nil {
		t.Fatalf("advance micro bar: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected one command for micro bar, got %d", len(cmds))
	}
	candidate, ok := cmds[0].(Candidate)
	if !ok || candidate.Interval != "1s" {
		t.Fatalf("unexpected micro bar candidate: %+v", cmds[0])
	}
}

func TestEngineCapsBarHistoryPerSymbolInterval(t *testing.T) {
	strategy := &stubStrategy{signal: core.Signal{Side: core.Flat}}
	engine := NewEngine(Config{ArmingState: ArmingSafe, Strategy: strategy})
	for i := 0; i < maxBarHistoryPerKey+25; i++ {
		_, err := engine.Advance(market.BarClosedEvent{SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(int64(1710000000+i), 0), Close: 62000 + float64(i)})
		if err != nil {
			t.Fatalf("advance bar %d: %v", i, err)
		}
	}
	if got := len(engine.bars["BTCUSDT:1m"]); got != maxBarHistoryPerKey {
		t.Fatalf("unexpected history size: %d", got)
	}
}

type stubStrategy struct {
	calls  int
	bars   []core.Bar
	signal core.Signal
}

func (strategy *stubStrategy) OnBar(symbol, interval string, bars []core.Bar) core.Signal {
	strategy.calls++
	strategy.bars = append([]core.Bar(nil), bars...)
	return strategy.signal
}

type fakeExchange struct {
	PlaceCalls int
	LastReq    bitget.PlaceOrderRequest
}

func (exchange *fakeExchange) PlaceOrder(req bitget.PlaceOrderRequest) error {
	exchange.PlaceCalls++
	exchange.LastReq = req
	return nil
}
