package market

import "time"

type PrivateFeed interface {
	Decode(raw []byte) ([]MarketEvent, error)
}

type PrivateDecoder func(raw []byte) ([]MarketEvent, error)

func (decoder PrivateDecoder) Decode(raw []byte) ([]MarketEvent, error) {
	return decoder(raw)
}

type OrderEvent struct {
	EventIDValue string
	SymbolValue  string
	Venue        string `json:"venue,omitempty"`
	MarketType   string `json:"market_type,omitempty"`
	Ts           time.Time
	ClientOID    string
	OrderID      string
	Status       string
	Size         float64
	Price        float64
	KindValue    string
}

func (event OrderEvent) EventID() string      { return event.EventIDValue }
func (event OrderEvent) Symbol() string       { return event.SymbolValue }
func (event OrderEvent) EventTime() time.Time { return event.Ts }
func (event OrderEvent) Kind() string         { return event.KindValue }

type PositionEvent struct {
	EventIDValue string
	SymbolValue  string
	Venue        string `json:"venue,omitempty"`
	MarketType   string `json:"market_type,omitempty"`
	Ts           time.Time
	Qty          float64
	KindValue    string
}

func (event PositionEvent) EventID() string      { return event.EventIDValue }
func (event PositionEvent) Symbol() string       { return event.SymbolValue }
func (event PositionEvent) EventTime() time.Time { return event.Ts }
func (event PositionEvent) Kind() string {
	if event.KindValue == "" {
		return "position_snapshot"
	}
	return event.KindValue
}

type AccountEvent struct {
	EventIDValue string
	SymbolValue  string
	Venue        string `json:"venue,omitempty"`
	MarketType   string `json:"market_type,omitempty"`
	Ts           time.Time
	MarginCoin   string
	Available    float64
	Equity       float64
	USDTEq       float64
	UnrealizedPL float64
	KindValue    string
}

func (event AccountEvent) EventID() string      { return event.EventIDValue }
func (event AccountEvent) Symbol() string       { return event.SymbolValue }
func (event AccountEvent) EventTime() time.Time { return event.Ts }
func (event AccountEvent) Kind() string {
	if event.KindValue == "" {
		return "account_snapshot"
	}
	return event.KindValue
}
