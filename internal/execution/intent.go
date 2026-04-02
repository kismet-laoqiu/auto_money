package execution

import (
	"encoding/json"
	"fmt"
	"time"

	"quantlab/internal/exchange/bitget"
	"quantlab/internal/trader"
)

type EntryIntentEvent = trader.EntryIntentEvent

type ReconcileEvent struct {
	EventIDValue   string    `json:"event_id"`
	SymbolValue    string    `json:"symbol"`
	Ts             time.Time `json:"ts"`
	IntentEventID  string    `json:"intent_event_id"`
	ClientOID      string    `json:"client_oid"`
	OrderID        string    `json:"order_id"`
	ExecutionVenue string    `json:"execution_venue"`
	Status         string    `json:"status"`
	Size           float64   `json:"size"`
	PriceAvg       float64   `json:"price_avg"`
	ReduceOnly     bool      `json:"reduce_only"`
}

func (event ReconcileEvent) EventID() string      { return event.EventIDValue }
func (event ReconcileEvent) Symbol() string       { return event.SymbolValue }
func (event ReconcileEvent) EventTime() time.Time { return event.Ts }
func (event ReconcileEvent) Kind() string         { return "execution.reconciled" }

func DecodeEntryIntentEvent(payload []byte) (EntryIntentEvent, error) {
	var event EntryIntentEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return EntryIntentEvent{}, err
	}
	return event, nil
}

func BuildReconcileEvent(intent EntryIntentEvent, detail bitget.OrderDetail) ReconcileEvent {
	return ReconcileEvent{
		EventIDValue:   fmt.Sprintf("reconcile:%s:%s:%d", intent.SymbolValue, firstNonEmpty(detail.ClientOID, intent.ClientOID), intent.Ts.UnixNano()),
		SymbolValue:    intent.SymbolValue,
		Ts:             time.Now().UTC(),
		IntentEventID:  intent.EventIDValue,
		ClientOID:      firstNonEmpty(detail.ClientOID, intent.ClientOID),
		OrderID:        detail.OrderID,
		ExecutionVenue: firstNonEmpty(intent.ExecutionVenue, "bitget"),
		Status:         detail.Status,
		Size:           detail.Size,
		PriceAvg:       detail.PriceAvg,
		ReduceOnly:     detail.ReduceOnly,
	}
}
