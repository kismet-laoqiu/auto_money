package insights

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlitepkg "quantlab/internal/store/sqlite"
)

func TestRuntimeAppendsAlertsToPlatformEventLog(t *testing.T) {
	runtime := NewRuntime(RuntimeConfig{
		Service: &serviceStub{},
		Store: &eventStoreStub{},
	})

	alert := Event{
		EventIDValue: "insight:daily_signal:BTCUSDT:1d:1710000000:long",
		SymbolValue:  "BTCUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		AlertType:    AlertTypeDailySignal,
		Interval:     "1d",
		Title:        "BTCUSDT 日线买点",
		Summary:      "score=4.80 threshold=4.50",
	}
	runtime.cfg.Service = &serviceStub{alerts: []Event{alert}}
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	store := runtime.cfg.Store.(*eventStoreStub)
	if len(store.events) != 1 || store.events[0].EventIDValue != alert.EventIDValue {
		t.Fatalf("unexpected stored alerts: %+v", store.events)
	}
}

func TestRuntimeIgnoresDuplicateEventIDs(t *testing.T) {
	alert := Event{
		EventIDValue: "insight:market_alert:BTCUSDT:15m:1710000000:surge",
		SymbolValue:  "BTCUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		AlertType:    AlertTypeMarketAlert,
		Interval:     "15m",
		Title:        "BTCUSDT 15m 暴涨",
	}
	runtime := NewRuntime(RuntimeConfig{
		Service: &serviceStub{alerts: []Event{alert}},
		Store:   &eventStoreStub{err: errors.New("UNIQUE constraint failed: event_log.event_id")},
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("expected duplicate to be ignored, got %v", err)
	}
}

type serviceStub struct {
	alerts []Event
	err    error
}

func (service *serviceStub) DetectAlerts(context.Context) ([]Event, error) {
	if service.err != nil {
		return nil, service.err
	}
	return append([]Event(nil), service.alerts...), nil
}

type eventStoreStub struct {
	events []Event
	err    error
}

func (store *eventStoreStub) AppendEvent(_ context.Context, _ string, evt sqlitepkg.LogEvent, _ []byte) (int64, error) {
	if store.err != nil {
		return 0, store.err
	}
	alert, ok := evt.(Event)
	if !ok {
		return 0, errors.New("unexpected event type")
	}
	store.events = append(store.events, alert)
	return int64(len(store.events)), nil
}
