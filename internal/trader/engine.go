package trader

import (
	"quantlab/internal/core"
	"quantlab/internal/market"
)

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
		return engine.onBarClosed(event), nil
	default:
		return nil, nil
	}
}

func (engine *Engine) State() EngineState {
	return engine.state
}

func (engine *Engine) onBarClosed(event market.BarClosedEvent) []Command {
	bar := core.Bar{
		Time:   event.Ts,
		Open:   event.Open,
		High:   event.High,
		Low:    event.Low,
		Close:  event.Close,
		Volume: event.Volume,
	}
	key := event.SymbolValue + ":" + event.Interval
	engine.bars[key] = append(engine.bars[key], bar)
	if engine.strategy == nil {
		return nil
	}
	signal := engine.strategy.OnBar(event.SymbolValue, event.Interval, engine.bars[key])
	if signal.Side == core.Flat {
		return nil
	}
	return []Command{Candidate{
		Symbol:   event.SymbolValue,
		Interval: event.Interval,
		Ts:       event.Ts,
		Side:     signal.Side,
		Score:    signal.Score,
		Entry:    signal.Entry,
		Stop:     signal.Stop,
		Target:   signal.Target,
		Reasons:  append([]string(nil), signal.Reasons...),
	}}
}
