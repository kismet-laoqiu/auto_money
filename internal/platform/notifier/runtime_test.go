package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

func TestRuntimeSendsRiskNotificationAndSavesCursor(t *testing.T) {
	store := &runtimeStoreStub{
		events: []sqlitepkg.EventEnvelope{
			{
				Seq:        11,
				Source:     "trader",
				EventID:    "risk-1",
				Symbol:     "MSTRUSDT",
				Kind:       "risk.state_changed",
				ExchangeTS: time.Unix(1710000000, 0).UTC(),
				Payload: mustJSON(t, trader.RiskEvent{
					EventIDValue: "risk-1",
					SymbolValue:  "MSTRUSDT",
					Ts:           time.Unix(1710000000, 0).UTC(),
					From:         trader.ArmingArmed,
					To:           trader.ArmingHalted,
					Reason:       "manual flatten",
				}),
			},
		},
	}
	sink := &sinkStub{}
	runtime := NewRuntime(RuntimeConfig{
		Store:   store,
		Service: sink,
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if store.savedCursor != 11 {
		t.Fatalf("unexpected saved cursor: %d", store.savedCursor)
	}
	if len(sink.notifications) != 1 {
		t.Fatalf("unexpected notifications: %+v", sink.notifications)
	}
	if sink.notifications[0].Kind != KindRiskHalt || sink.notifications[0].Summary == "" {
		t.Fatalf("unexpected notification payload: %+v", sink.notifications[0])
	}
}

func TestRuntimeIgnoresUnknownKindsButStillAdvancesCursor(t *testing.T) {
	store := &runtimeStoreStub{
		events: []sqlitepkg.EventEnvelope{
			{
				Seq:        7,
				Source:     "market.public",
				EventID:    "trade-1",
				Symbol:     "MSTRUSDT",
				Kind:       "trade_tick",
				ExchangeTS: time.Unix(1710000000, 0).UTC(),
				Payload:    []byte(`{"symbol":"MSTRUSDT"}`),
			},
		},
	}
	sink := &sinkStub{}
	runtime := NewRuntime(RuntimeConfig{
		Store:   store,
		Service: sink,
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if store.savedCursor != 7 {
		t.Fatalf("unexpected saved cursor: %d", store.savedCursor)
	}
	if len(sink.notifications) != 0 {
		t.Fatalf("unexpected notifications: %+v", sink.notifications)
	}
}

func TestRuntimeWithSQLiteStorePersistsCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifier-runtime.db")
	store, err := sqlitepkg.NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	event := trader.RiskEvent{
		EventIDValue: "risk-sqlite",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		From:         trader.ArmingArmed,
		To:           trader.ArmingHalted,
		Reason:       "sqlite integration",
	}
	body := mustJSON(t, event)
	seq, err := store.AppendEvent(context.Background(), "trader", event, body)
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	sink := &sinkStub{}
	runtime := NewRuntime(RuntimeConfig{
		Store:   store,
		Service: sink,
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	cursor, err := store.LoadConsumerCursor(context.Background(), "notifierd")
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if cursor != seq {
		t.Fatalf("unexpected cursor: got=%d want=%d", cursor, seq)
	}
	if len(sink.notifications) != 1 {
		t.Fatalf("unexpected notifications: %+v", sink.notifications)
	}
}

func TestRuntimeWritesDeadLetterAfterRetryExhausted(t *testing.T) {
	store := &runtimeStoreStub{
		events: []sqlitepkg.EventEnvelope{
			{
				Seq:        13,
				Source:     "platform",
				EventID:    "promotion-1",
				Symbol:     "mstr-wave-fib",
				Kind:       "promotion.approved",
				ExchangeTS: time.Unix(1710000000, 0).UTC(),
				Payload: []byte(`{
					"title":"mstr-wave-fib v0.1.0 promotion approved",
					"summary":"promotion is live_active",
					"details":["promotion_id=promo-1"]
				}`),
			},
		},
	}
	runtime := NewRuntime(RuntimeConfig{
		Store:       store,
		Service:     errSink{err: errors.New("telegram timeout")},
		MaxAttempts: 3,
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if store.savedCursor != 13 {
		t.Fatalf("unexpected saved cursor: %d", store.savedCursor)
	}
	if len(store.appendedKinds) != 1 || store.appendedKinds[0] != "notification.dead_letter" {
		t.Fatalf("unexpected dead-letter events: %+v", store.appendedKinds)
	}
}

func TestRuntimeSendsInsightAlertFromPlatformEvent(t *testing.T) {
	store := &runtimeStoreStub{
		events: []sqlitepkg.EventEnvelope{
			{
				Seq:        21,
				Source:     "platform",
				EventID:    "insight:daily_signal:BTCUSDT:1d:1710000000:long",
				Symbol:     "BTCUSDT",
				Kind:       "insight.alert",
				ExchangeTS: time.Unix(1710000000, 0).UTC(),
				Payload: []byte(`{
					"alert_type":"daily_signal",
					"title":"BTCUSDT 日线买点",
					"summary":"score=4.82 threshold=4.60 close=81234.1",
					"details":["reasons=trend up, break retest"]
				}`),
			},
		},
	}
	sink := &sinkStub{}
	runtime := NewRuntime(RuntimeConfig{
		Store:   store,
		Service: sink,
	})
	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if len(sink.notifications) != 1 {
		t.Fatalf("unexpected notifications: %+v", sink.notifications)
	}
	if sink.notifications[0].Kind != KindDailySignal {
		t.Fatalf("unexpected insight notification: %+v", sink.notifications[0])
	}
}

type runtimeStoreStub struct {
	loadedCursor  int64
	savedCursor   int64
	events        []sqlitepkg.EventEnvelope
	appendedKinds []string
}

func (store *runtimeStoreStub) ListEventsAfter(_ context.Context, afterSeq int64, _ int, _ ...string) ([]sqlitepkg.EventEnvelope, error) {
	if afterSeq >= store.savedCursor {
		return store.events, nil
	}
	return store.events, nil
}

func (store *runtimeStoreStub) SaveConsumerCursor(_ context.Context, _ string, seq int64) error {
	store.savedCursor = seq
	return nil
}

func (store *runtimeStoreStub) LoadConsumerCursor(_ context.Context, _ string) (int64, error) {
	return store.loadedCursor, nil
}

func (store *runtimeStoreStub) AppendEvent(_ context.Context, _ string, evt sqlitepkg.LogEvent, _ []byte) (int64, error) {
	store.appendedKinds = append(store.appendedKinds, evt.Kind())
	return int64(len(store.appendedKinds)), nil
}

type sinkStub struct {
	notifications []Notification
}

func (sink *sinkStub) Notify(_ context.Context, notification Notification) error {
	sink.notifications = append(sink.notifications, notification)
	return nil
}

type errSink struct {
	err error
}

func (sink errSink) Notify(_ context.Context, _ Notification) error {
	return sink.err
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test payload: %v", err)
	}
	return body
}
