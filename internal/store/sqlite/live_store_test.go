package sqlite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"quantlab/internal/market"
	"quantlab/internal/trader"
)

func TestAppendEventAndCheckpointRoundTrip(t *testing.T) {
	store := newTestStore(t)
	evt := market.BarClosedEvent{EventIDValue: "evt-1", SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000000, 0)}
	raw := []byte(`{"close":"62000"}`)
	if _, err := store.AppendEvent(context.Background(), "market", evt, raw); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if got := readEventPayload(t, store, evt.EventID()); got != string(raw) {
		t.Fatalf("unexpected logged payload: %s", got)
	}
	if err := store.SaveCheckpoint(context.Background(), "btc", trader.EngineState{ArmingState: trader.ArmingSafe}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	got, err := store.LoadCheckpoint(context.Background(), "btc")
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if got.ArmingState != trader.ArmingSafe {
		t.Fatalf("unexpected checkpoint: %+v", got)
	}
}

func TestAppendEventAllowsNilPayload(t *testing.T) {
	store := newTestStore(t)
	evt := market.BarClosedEvent{EventIDValue: "evt-2", SymbolValue: "BTCUSDT", Interval: "1m", Ts: time.Unix(1710000001, 0)}
	if _, err := store.AppendEvent(context.Background(), "market", evt, nil); err != nil {
		t.Fatalf("append event with nil payload: %v", err)
	}
	if got := readEventPayload(t, store, evt.EventID()); got != "null" {
		t.Fatalf("unexpected nil payload encoding: %s", got)
	}
}

func TestAppendEventReturnsMonotonicSeqAndListEventsAfter(t *testing.T) {
	store := newTestStore(t)
	first, err := store.AppendEvent(context.Background(), "market", market.BarClosedEvent{
		EventIDValue: "evt-1",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
	}, []byte(`{"close":"62000"}`))
	if err != nil {
		t.Fatalf("append first: %v", err)
	}
	second, err := store.AppendEvent(context.Background(), "trader", market.BarClosedEvent{
		EventIDValue: "evt-2",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000060, 0),
	}, []byte(`{"close":"62050"}`))
	if err != nil {
		t.Fatalf("append second: %v", err)
	}
	if second <= first {
		t.Fatalf("seq must increase: first=%d second=%d", first, second)
	}

	events, err := store.ListEventsAfter(context.Background(), first, 10, "trader")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Seq != second || events[0].Source != "trader" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestConsumerCursorRoundTrip(t *testing.T) {
	store := newTestStore(t)
	if err := store.SaveConsumerCursor(context.Background(), "traderd", 42); err != nil {
		t.Fatalf("save cursor: %v", err)
	}
	got, err := store.LoadConsumerCursor(context.Background(), "traderd")
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if got != 42 {
		t.Fatalf("unexpected cursor: %d", got)
	}
}

func TestListConsumerCursorsAndLastSeq(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	for index, consumer := range []string{"marketd", "traderd"} {
		if err := store.SaveConsumerCursor(ctx, consumer, int64(index+1)); err != nil {
			t.Fatalf("save cursor %s: %v", consumer, err)
		}
	}
	if _, err := store.AppendEvent(ctx, "market.public", market.BarClosedEvent{
		EventIDValue: "evt-last-seq",
		SymbolValue:  "BTCUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0),
	}, []byte(`{"close":"62000"}`)); err != nil {
		t.Fatalf("append event: %v", err)
	}

	lastSeq, err := store.LastSeq(ctx)
	if err != nil {
		t.Fatalf("last seq: %v", err)
	}
	if lastSeq != 1 {
		t.Fatalf("unexpected last seq: %d", lastSeq)
	}

	cursors, err := store.ListConsumerCursors(ctx)
	if err != nil {
		t.Fatalf("list cursors: %v", err)
	}
	if len(cursors) != 2 {
		t.Fatalf("unexpected cursor count: %d", len(cursors))
	}
	if cursors[0].ConsumerKey != "marketd" || cursors[1].ConsumerKey != "traderd" {
		t.Fatalf("unexpected cursor order: %+v", cursors)
	}
}

func TestListRecentEventsReturnsNewestFirst(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	for index, eventID := range []string{"evt-1", "evt-2", "evt-3"} {
		if _, err := store.AppendEvent(ctx, "market.public", market.BarClosedEvent{
			EventIDValue: eventID,
			SymbolValue:  "BTCUSDT",
			Interval:     "1m",
			Ts:           time.Unix(1710000000+int64(index), 0),
		}, []byte(`{}`)); err != nil {
			t.Fatalf("append %s: %v", eventID, err)
		}
	}

	events, err := store.ListRecentEvents(ctx, 2)
	if err != nil {
		t.Fatalf("list recent events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("unexpected recent event count: %d", len(events))
	}
	if events[0].EventID != "evt-3" || events[1].EventID != "evt-2" {
		t.Fatalf("unexpected event order: %+v", events)
	}
}

func TestListEventsAfterAcceptsTextPayload(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.db.Exec(`
		INSERT INTO event_log (source, event_id, symbol, event_kind, exchange_ts, received_ts, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "trader", "evt-text-payload", "MSTRUSDT", "risk.state_changed", "2026-03-29T04:45:00Z", "2026-03-29T04:45:00Z", `{"event_id":"evt-text-payload","symbol":"MSTRUSDT"}`); err != nil {
		t.Fatalf("seed text payload event: %v", err)
	}

	events, err := store.ListEventsAfter(context.Background(), 0, 10, "trader")
	if err != nil {
		t.Fatalf("list events after: %v", err)
	}
	if len(events) != 1 || string(events[0].Payload) != `{"event_id":"evt-text-payload","symbol":"MSTRUSDT"}` {
		t.Fatalf("unexpected text payload events: %+v", events)
	}
}

func TestListLatestPositionsReturnsNewestPerSymbol(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	appendPositionEvent(t, ctx, store, market.PositionEvent{
		EventIDValue: "pos-1",
		SymbolValue:  "BTCUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		Qty:          0.01,
		KindValue:    "position_snapshot",
	})
	appendPositionEvent(t, ctx, store, market.PositionEvent{
		EventIDValue: "pos-2",
		SymbolValue:  "BTCUSDT",
		Ts:           time.Unix(1710000100, 0).UTC(),
		Qty:          0.04,
		KindValue:    "position_update",
	})
	appendPositionEvent(t, ctx, store, market.PositionEvent{
		EventIDValue: "pos-3",
		SymbolValue:  "ETHUSDT",
		Ts:           time.Unix(1710000200, 0).UTC(),
		Qty:          0,
		KindValue:    "position_snapshot",
	})

	positions, err := store.ListLatestPositions(ctx)
	if err != nil {
		t.Fatalf("list latest positions: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("unexpected position count: %d", len(positions))
	}
	symbols := []string{positions[0].Symbol, positions[1].Symbol}
	if !slices.Equal(symbols, []string{"BTCUSDT", "ETHUSDT"}) {
		t.Fatalf("unexpected position symbols: %+v", positions)
	}
	if positions[0].Qty != 0.04 || positions[0].Kind != "position_update" {
		t.Fatalf("unexpected btc position: %+v", positions[0])
	}
}

func TestListLatestOrdersReturnsNewestFirst(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	appendOrderEvent(t, ctx, store, market.OrderEvent{
		EventIDValue: "order-1",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000000, 0).UTC(),
		ClientOID:    "client-1",
		OrderID:      "1001",
		Status:       "filled",
		Size:         0.04,
		Price:        126.33,
		KindValue:    "order_fill",
	})
	appendOrderEvent(t, ctx, store, market.OrderEvent{
		EventIDValue: "order-2",
		SymbolValue:  "MSTRUSDT",
		Ts:           time.Unix(1710000100, 0).UTC(),
		ClientOID:    "client-2",
		OrderID:      "1002",
		Status:       "live",
		Size:         0.04,
		Price:        126.50,
		KindValue:    "order_update",
	})

	orders, err := store.ListLatestOrders(ctx, 10)
	if err != nil {
		t.Fatalf("list latest orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("unexpected order count: %d", len(orders))
	}
	if orders[0].OrderID != "1002" || orders[1].OrderID != "1001" {
		t.Fatalf("unexpected order order: %+v", orders)
	}
}

func TestNewStoreCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "demo", "live-state.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store instance")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat db path: %v", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "live-state.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func readEventPayload(t *testing.T, store *Store, eventID string) string {
	t.Helper()
	var body []byte
	err := store.db.QueryRow(`SELECT payload_json FROM event_log WHERE event_id = ?`, eventID).Scan(&body)
	if err != nil {
		t.Fatalf("read event payload: %v", err)
	}
	return string(body)
}

func appendPositionEvent(t *testing.T, ctx context.Context, store *Store, event market.PositionEvent) {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal position event: %v", err)
	}
	if _, err := store.AppendEvent(ctx, "market.private", event, body); err != nil {
		t.Fatalf("append position event: %v", err)
	}
}

func appendOrderEvent(t *testing.T, ctx context.Context, store *Store, event market.OrderEvent) {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal order event: %v", err)
	}
	if _, err := store.AppendEvent(ctx, "market.private", event, body); err != nil {
		t.Fatalf("append order event: %v", err)
	}
}
