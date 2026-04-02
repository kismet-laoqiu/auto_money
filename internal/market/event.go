package market

import "time"

type MarketEvent interface {
	EventID() string
	Symbol() string
	EventTime() time.Time
	Kind() string
}

type TradeTickEvent struct {
	EventIDValue string
	SymbolValue  string
	Venue        string `json:"venue,omitempty"`
	MarketType   string `json:"market_type,omitempty"`
	Ts           time.Time
	Price        float64
	Size         float64
	Side         string
}

func (event TradeTickEvent) EventID() string      { return event.EventIDValue }
func (event TradeTickEvent) Symbol() string       { return event.SymbolValue }
func (event TradeTickEvent) EventTime() time.Time { return event.Ts }
func (event TradeTickEvent) Kind() string         { return "trade_tick" }

type BarClosedEvent struct {
	EventIDValue string
	SymbolValue  string
	Venue        string `json:"venue,omitempty"`
	MarketType   string `json:"market_type,omitempty"`
	Interval     string
	Ts           time.Time
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       float64
}

func (event BarClosedEvent) EventID() string      { return event.EventIDValue }
func (event BarClosedEvent) Symbol() string       { return event.SymbolValue }
func (event BarClosedEvent) EventTime() time.Time { return event.Ts }
func (event BarClosedEvent) Kind() string         { return "bar_closed" }

type MicroBarClosedEvent struct {
	EventIDValue string
	SymbolValue  string
	Venue        string `json:"venue,omitempty"`
	MarketType   string `json:"market_type,omitempty"`
	OpenedAt     time.Time
	ClosedAt     time.Time
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       float64
}

func (event MicroBarClosedEvent) EventID() string      { return event.EventIDValue }
func (event MicroBarClosedEvent) Symbol() string       { return event.SymbolValue }
func (event MicroBarClosedEvent) EventTime() time.Time { return event.ClosedAt }
func (event MicroBarClosedEvent) Kind() string         { return "micro_bar_closed" }
