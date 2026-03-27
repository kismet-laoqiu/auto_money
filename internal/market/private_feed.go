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
