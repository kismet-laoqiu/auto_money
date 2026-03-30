package execution

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

func TestRuntimeExecutesIntentAndAppendsReconcileEvent(t *testing.T) {
	ctx := context.Background()
	intent := trader.EntryIntentEvent{
		EventIDValue: "intent:MSTRUSDT:1m:1710000000000",
		SymbolValue:  "MSTRUSDT",
		Interval:     "1m",
		Ts:           time.Unix(1710000000, 0).UTC(),
		Side:         core.Long,
		Score:        4.2,
		Entry:        125,
		Stop:         120,
		Target:       135,
		ProductType:  "USDT-FUTURES",
		MarginMode:   "isolated",
		MarginCoin:   "USDT",
		Size:         "0.04",
		Leverage:     "3",
		ClientOID:    "cid-1",
	}
	payload, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("marshal intent: %v", err)
	}
	store := &memoryStore{
		events: []sqlitepkg.EventEnvelope{{
			Seq:        7,
			Source:     "trader",
			EventID:    intent.EventID(),
			Symbol:     intent.Symbol(),
			Kind:       intent.Kind(),
			ExchangeTS: intent.EventTime(),
			ReceivedTS: intent.EventTime(),
			Payload:    payload,
		}},
		cursors: map[string]int64{},
	}
	client := &fakeExchangeClient{
		orderResult: bitget.OrderResult{OrderID: "oid-1", ClientOID: "cid-1"},
		details: []bitget.OrderDetail{{
			OrderID:    "oid-1",
			ClientOID:  "cid-1",
			Status:     "filled",
			Size:       0.04,
			PriceAvg:   125.1,
			ReduceOnly: false,
		}},
	}
	runtime := NewRuntime(RuntimeConfig{
		Store:             store,
		Exchange:          client,
		OrderPollInterval: time.Nanosecond,
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if len(client.leverageReqs) != 1 {
		t.Fatalf("expected one leverage request, got %+v", client.leverageReqs)
	}
	if got := client.leverageReqs[0]; got.Symbol != "MSTRUSDT" || got.Leverage != "3" || got.HoldSide != "long" {
		t.Fatalf("unexpected leverage request: %+v", got)
	}
	if len(client.orderReqs) != 1 {
		t.Fatalf("expected one place-order request, got %+v", client.orderReqs)
	}
	if got := client.orderReqs[0]; got.Side != "buy" || got.TradeSide != "open" || got.Size != "0.04" || got.ClientOID != "cid-1" {
		t.Fatalf("unexpected order request: %+v", got)
	}
	if len(store.appended) != 1 {
		t.Fatalf("expected one reconcile event, got %+v", store.appended)
	}
	if store.appended[0].source != "execution" || store.appended[0].event.Kind() != "execution.reconciled" {
		t.Fatalf("unexpected appended event: %+v", store.appended[0])
	}
	var reconcile ReconcileEvent
	if err := json.Unmarshal(store.appended[0].raw, &reconcile); err != nil {
		t.Fatalf("decode reconcile payload: %v", err)
	}
	if reconcile.OrderID != "oid-1" || reconcile.Status != "filled" || reconcile.ClientOID != "cid-1" {
		t.Fatalf("unexpected reconcile payload: %+v", reconcile)
	}
	if store.cursors["execd"] != 7 {
		t.Fatalf("unexpected cursor: %+v", store.cursors)
	}
}

func TestRuntimeIgnoresNonIntentEvents(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{
		events: []sqlitepkg.EventEnvelope{{
			Seq:        1,
			Source:     "trader",
			EventID:    "candidate-1",
			Symbol:     "MSTRUSDT",
			Kind:       "candidate.created",
			ExchangeTS: time.Unix(1710000000, 0).UTC(),
			ReceivedTS: time.Unix(1710000000, 0).UTC(),
			Payload:    []byte(`{"event_id":"candidate-1"}`),
		}},
		cursors: map[string]int64{},
	}
	client := &fakeExchangeClient{}
	runtime := NewRuntime(RuntimeConfig{
		Store:             store,
		Exchange:          client,
		OrderPollInterval: time.Nanosecond,
	})

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	if len(client.orderReqs) != 0 || len(store.appended) != 0 {
		t.Fatalf("non-intent event must be ignored: orderReqs=%+v appended=%+v", client.orderReqs, store.appended)
	}
}

func TestRuntimeFlattenSymbolUsesFetchedPositionMode(t *testing.T) {
	ctx := context.Background()
	client := &fakeExchangeClient{
		position: bitget.SinglePositionSnapshot{
			Qty:  0.04,
			Mode: "one_way_mode",
		},
		orderResult: bitget.OrderResult{OrderID: "oid-close", ClientOID: "close-1"},
		details: []bitget.OrderDetail{{
			OrderID:    "oid-close",
			ClientOID:  "close-1",
			Status:     "filled",
			Size:       0.04,
			PriceAvg:   124.9,
			ReduceOnly: true,
		}},
	}
	runtime := NewRuntime(RuntimeConfig{
		Exchange:          client,
		OrderPollInterval: time.Nanosecond,
	})

	if err := runtime.FlattenSymbol(ctx, "MSTRUSDT"); err != nil {
		t.Fatalf("flatten symbol: %v", err)
	}
	if len(client.orderReqs) != 1 {
		t.Fatalf("expected one close order, got %+v", client.orderReqs)
	}
	got := client.orderReqs[0]
	if got.Side != "sell" || got.ReduceOnly != "YES" || got.TradeSide != "" || got.Size != "0.04" {
		t.Fatalf("unexpected close request: %+v", got)
	}
}

type appendedEvent struct {
	source string
	event  sqlitepkg.LogEvent
	raw    []byte
}

type memoryStore struct {
	events   []sqlitepkg.EventEnvelope
	cursors  map[string]int64
	appended []appendedEvent
}

func (store *memoryStore) AppendEvent(_ context.Context, source string, evt sqlitepkg.LogEvent, raw []byte) (int64, error) {
	store.appended = append(store.appended, appendedEvent{
		source: source,
		event:  evt,
		raw:    append([]byte(nil), raw...),
	})
	return int64(len(store.events) + len(store.appended)), nil
}

func (store *memoryStore) ListEventsAfter(_ context.Context, afterSeq int64, limit int, sources ...string) ([]sqlitepkg.EventEnvelope, error) {
	filter := map[string]bool{}
	for _, source := range sources {
		filter[source] = true
	}
	out := make([]sqlitepkg.EventEnvelope, 0, len(store.events))
	for _, event := range store.events {
		if event.Seq <= afterSeq {
			continue
		}
		if len(filter) != 0 && !filter[event.Source] {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (store *memoryStore) SaveConsumerCursor(_ context.Context, consumer string, seq int64) error {
	store.cursors[consumer] = seq
	return nil
}

func (store *memoryStore) LoadConsumerCursor(_ context.Context, consumer string) (int64, error) {
	return store.cursors[consumer], nil
}

type fakeExchangeClient struct {
	leverageReqs []bitget.SetLeverageRequest
	orderReqs    []bitget.PlaceOrderRequest
	detailReqs   []bitget.OrderDetailRequest
	positionReqs []string
	orderResult  bitget.OrderResult
	details      []bitget.OrderDetail
	position     bitget.SinglePositionSnapshot
}

func (client *fakeExchangeClient) SetLeverage(_ context.Context, req bitget.SetLeverageRequest) (bitget.LeverageSetting, error) {
	client.leverageReqs = append(client.leverageReqs, req)
	return bitget.LeverageSetting{
		Symbol:       req.Symbol,
		MarginCoin:   req.MarginCoin,
		LongLeverage: req.Leverage,
		MarginMode:   "isolated",
	}, nil
}

func (client *fakeExchangeClient) PlaceOrder(_ context.Context, req bitget.PlaceOrderRequest) (bitget.OrderResult, error) {
	client.orderReqs = append(client.orderReqs, req)
	if client.orderResult.OrderID == "" {
		client.orderResult = bitget.OrderResult{OrderID: "oid-1", ClientOID: req.ClientOID}
	}
	return client.orderResult, nil
}

func (client *fakeExchangeClient) GetOrderDetail(_ context.Context, req bitget.OrderDetailRequest) (bitget.OrderDetail, error) {
	client.detailReqs = append(client.detailReqs, req)
	if len(client.details) == 0 {
		return bitget.OrderDetail{OrderID: req.OrderID, ClientOID: req.ClientOID, Status: "filled"}, nil
	}
	detail := client.details[0]
	client.details = client.details[1:]
	return detail, nil
}

func (client *fakeExchangeClient) FetchSinglePosition(_ context.Context, symbol, _, _ string) (bitget.SinglePositionSnapshot, error) {
	client.positionReqs = append(client.positionReqs, symbol)
	return client.position, nil
}
