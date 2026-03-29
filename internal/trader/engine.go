package trader

import (
	"quantlab/internal/core"
	"quantlab/internal/market"
)

const maxBarHistoryPerKey = 2048

type Strategy interface {
	OnBar(symbol, interval string, bars []core.Bar) core.Signal
}

type Config struct {
	ArmingState ArmingState
	Strategy    Strategy
}

type Engine struct {
	state    EngineState
	strategy Strategy
	bars     map[string][]core.Bar
}

func NewEngine(cfg Config) *Engine {
	state := EngineState{ArmingState: cfg.ArmingState, Symbols: map[string]SymbolState{}}
	if state.ArmingState == "" {
		state.ArmingState = ArmingSafe
	}
	return &Engine{
		state:    state,
		strategy: cfg.Strategy,
		bars:     map[string][]core.Bar{},
	}
}

func (engine *Engine) Advance(evt market.MarketEvent) ([]Command, error) {
	switch event := evt.(type) {
	case market.BarClosedEvent:
		return engine.advanceBar(event.SymbolValue, event.Interval, core.Bar{Time: event.Ts, Open: event.Open, High: event.High, Low: event.Low, Close: event.Close, Volume: event.Volume})
	case market.MicroBarClosedEvent:
		return engine.advanceBar(event.SymbolValue, "1s", core.Bar{Time: event.ClosedAt, Open: event.Open, High: event.High, Low: event.Low, Close: event.Close, Volume: event.Volume})
	default:
		return nil, nil
	}
}

func (engine *Engine) State() EngineState {
	return cloneEngineState(engine.state)
}

func (engine *Engine) Restore(state EngineState) {
	engine.state = cloneEngineState(state)
	if engine.state.ArmingState == "" {
		engine.state.ArmingState = ArmingSafe
	}
}

func (engine *Engine) advanceBar(symbol, interval string, bar core.Bar) ([]Command, error) {
	key := symbol + ":" + interval
	history := append(engine.bars[key], bar)
	if len(history) > maxBarHistoryPerKey {
		history = history[len(history)-maxBarHistoryPerKey:]
	}
	engine.bars[key] = history
	if engine.strategy == nil {
		return nil, nil
	}
	signal := engine.strategy.OnBar(symbol, interval, history)
	if signal.Side == core.Flat {
		return nil, nil
	}
	return []Command{Candidate{Symbol: symbol, Interval: interval, Ts: bar.Time, Side: signal.Side, Score: signal.Score, Entry: signal.Entry, Stop: signal.Stop, Target: signal.Target, Reasons: append([]string(nil), signal.Reasons...)}}, nil
}

func cloneEngineState(state EngineState) EngineState {
	cloned := EngineState{
		ArmingState: state.ArmingState,
		Symbols:     map[string]SymbolState{},
	}
	for symbol, snapshot := range state.Symbols {
		cloned.Symbols[symbol] = snapshot
	}
	return cloned
}
