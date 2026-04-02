package common

import "time"

type MarketType string

const (
	MarketTypePerp MarketType = "perp"
	MarketTypeSpot MarketType = "spot"
)

type Instrument struct {
	Venue           string
	MarketType      MarketType
	CanonicalSymbol string
	VenueSymbol     string
	AssetID         string
}

type Candle struct {
	Venue           string
	MarketType      MarketType
	CanonicalSymbol string
	VenueSymbol     string
	Interval        string
	OpenTime        time.Time
	CloseTime       time.Time
	Open            float64
	High            float64
	Low             float64
	Close           float64
	Volume          float64
	TradeCount      int
}

type PriceSnapshot struct {
	Venue           string
	MarketType      MarketType
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	LastPrice       float64
	MarkPrice       float64
}

type FundingRate struct {
	Venue           string
	MarketType      MarketType
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	Rate            float64
}

type OpenInterest struct {
	Venue           string
	MarketType      MarketType
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	Value           float64
	ValueUSD        float64
}

type ContractSpec struct {
	Venue           string
	MarketType      MarketType
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	MinTradeSize    float64
	SizeStep        float64
	PriceStep       float64
	MaxLeverage     int
}

type LeaderFill struct {
	Venue           string
	MarketType      MarketType
	LeaderID        string
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	Side            string
	Price           float64
	Size            float64
	Fee             float64
	RealizedPnL     float64
	VenueOrderID    string
	VenueTradeID    string
}

type LeaderStateSnapshot struct {
	Venue           string
	MarketType      MarketType
	LeaderID        string
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	PositionSize    float64
	Leverage        float64
	EntryPrice      float64
	AccountValue    float64
}

type OrderUpdate struct {
	Venue           string
	MarketType      MarketType
	LeaderID        string
	CanonicalSymbol string
	VenueSymbol     string
	Timestamp       time.Time
	Side            string
	Status          string
	Price           float64
	Size            float64
	FilledSize      float64
	ReduceOnly      bool
	ClientOrderID   string
	VenueOrderID    string
}
