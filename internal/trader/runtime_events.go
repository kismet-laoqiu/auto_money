package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/market"
)

type EventLog interface {
	EventID() string
	Symbol() string
	EventTime() time.Time
	Kind() string
}

type EventEnvelope struct {
	Seq        int64
	Source     string
	EventID    string
	Symbol     string
	Kind       string
	ExchangeTS time.Time
	ReceivedTS time.Time
	Payload    json.RawMessage
}

type RuntimeStore interface {
	AppendEvent(ctx context.Context, source string, evt EventLog, raw []byte) (int64, error)
	ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]EventEnvelope, error)
	SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error
	LoadConsumerCursor(ctx context.Context, consumer string) (int64, error)
	SaveCheckpoint(ctx context.Context, shard string, state EngineState) error
	LoadCheckpoint(ctx context.Context, shard string) (EngineState, error)
}

type CandidateEvent struct {
	EventIDValue string    `json:"event_id"`
	SymbolValue  string    `json:"symbol"`
	Interval     string    `json:"interval"`
	Ts           time.Time `json:"ts"`
	Side         core.Side `json:"side"`
	Score        float64   `json:"score"`
	Entry        float64   `json:"entry"`
	Stop         float64   `json:"stop"`
	Target       float64   `json:"target"`
	Reasons      []string  `json:"reasons,omitempty"`
}

func (event CandidateEvent) EventID() string      { return event.EventIDValue }
func (event CandidateEvent) Symbol() string       { return event.SymbolValue }
func (event CandidateEvent) EventTime() time.Time { return event.Ts }
func (event CandidateEvent) Kind() string         { return "candidate.created" }

type RiskEvent struct {
	EventIDValue string      `json:"event_id"`
	SymbolValue  string      `json:"symbol"`
	Ts           time.Time   `json:"ts"`
	From         ArmingState `json:"from"`
	To           ArmingState `json:"to"`
	Reason       string      `json:"reason"`
}

func (event RiskEvent) EventID() string      { return event.EventIDValue }
func (event RiskEvent) Symbol() string       { return event.SymbolValue }
func (event RiskEvent) EventTime() time.Time { return event.Ts }
func (event RiskEvent) Kind() string         { return "risk.state_changed" }

func BuildCandidateEvent(candidate Candidate) CandidateEvent {
	return CandidateEvent{
		EventIDValue: fmt.Sprintf("candidate:%s:%s:%d", candidate.Symbol, candidate.Interval, candidate.Ts.UnixNano()),
		SymbolValue:  candidate.Symbol,
		Interval:     candidate.Interval,
		Ts:           candidate.Ts,
		Side:         candidate.Side,
		Score:        candidate.Score,
		Entry:        candidate.Entry,
		Stop:         candidate.Stop,
		Target:       candidate.Target,
		Reasons:      append([]string(nil), candidate.Reasons...),
	}
}

func DecodeMarketEvent(env EventEnvelope) (market.MarketEvent, error) {
	switch env.Kind {
	case "trade_tick":
		var event market.TradeTickEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, fmt.Errorf("decode trade tick: %w", err)
		}
		return event, nil
	case "bar_closed":
		var event market.BarClosedEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, fmt.Errorf("decode bar closed: %w", err)
		}
		return event, nil
	case "micro_bar_closed":
		var event market.MicroBarClosedEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, fmt.Errorf("decode micro bar closed: %w", err)
		}
		return event, nil
	case "order_update", "order_fill":
		var event market.OrderEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, fmt.Errorf("decode order event: %w", err)
		}
		return event, nil
	case "position_snapshot", "position_update":
		var event market.PositionEvent
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return nil, fmt.Errorf("decode position event: %w", err)
		}
		return event, nil
	default:
		return nil, nil
	}
}
