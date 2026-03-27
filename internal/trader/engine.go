package trader

import (
	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
)

const maxBarHistoryPerKey = 2048

type Strategy interface {
	OnBar(symbol, interval string, bars []core.Bar) core.Signal
}

type Exchange interface {
	PlaceOrder(req bitget.PlaceOrderRequest) error
}

type Config struct {
	ArmingState ArmingState
	Strategy    Strategy
	Exchange    Exchange
}

type Engine struct {
	state    EngineState
	strategy Strategy
	exchange Exchange
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
		exchange: cfg.Exchange,
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
	return engine.state
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
	commands := []Command{Candidate{Symbol: symbol, Interval: interval, Ts: bar.Time, Side: signal.Side, Score: signal.Score, Entry: signal.Entry, Stop: signal.Stop, Target: signal.Target, Reasons: append([]string(nil), signal.Reasons...)}}
	if engine.exchange == nil || engine.state.ArmingState != ArmingArmed {
		return commands, nil
	}
	side := "buy"
	if signal.Side == core.Short {
		side = "sell"
	}
	err := engine.exchange.PlaceOrder(bitget.PlaceOrderRequest{Symbol: symbol, ProductType: "USDT-FUTURES", MarginMode: "isolated", MarginCoin: "USDT", Side: side, OrderType: "market", Size: "1"})
	return commands, err
}
