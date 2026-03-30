package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

const (
	realSymbol      = "MSTRUSDT"
	realProductType = "USDT-FUTURES"
	realMarginMode  = "isolated"
	realMarginCoin  = "USDT"
)

func TestRealExecutionRuntimeIntentLifecycle(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 to run real Bitget execution test")
	}
	creds := bitget.PrivateCredentials{
		Key:        os.Getenv("BITGET_API_KEY"),
		Secret:     os.Getenv("BITGET_API_SECRET"),
		Passphrase: os.Getenv("BITGET_PASSPHRASE"),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Fatal("missing Bitget credentials in environment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client := bitget.NewPrivateClient("", creds)
	store, err := sqlitepkg.NewStore(t.TempDir() + "/live.db")
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	runtime := NewRuntime(RuntimeConfig{
		Store:             store,
		Exchange:          client,
		ProductType:       realProductType,
		MarginMode:        realMarginMode,
		MarginCoin:        realMarginCoin,
		OrderPollInterval: time.Second,
	})
	if err := ensureFlat(ctx, runtime, client); err != nil {
		t.Fatalf("preflight ensure flat: %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		if err := ensureFlat(cleanupCtx, runtime, client); err != nil {
			t.Errorf("cleanup ensure flat: %v", err)
		}
	}()

	lastPrice, err := client.FetchTickerPrice(ctx, realSymbol, realProductType)
	if err != nil {
		t.Fatalf("fetch ticker price: %v", err)
	}
	size := trader.ComputeIntentSizeText(lastPrice)
	t.Logf("real execd preflight: symbol=%s last_price=%.4f size=%s", realSymbol, lastPrice, size)
	now := time.Now().UTC()
	clientOID := fmt.Sprintf("ql-execd-real-open-%d", now.UnixMilli())

	intent := trader.EntryIntentEvent{
		EventIDValue: fmt.Sprintf("intent:real-execd:%d", now.UnixNano()),
		SymbolValue:  realSymbol,
		Interval:     "1m",
		Ts:           now,
		Side:         core.Long,
		Score:        4.2,
		Entry:        lastPrice,
		Stop:         lastPrice * 0.98,
		Target:       lastPrice * 1.04,
		ProductType:  realProductType,
		MarginMode:   realMarginMode,
		MarginCoin:   realMarginCoin,
		Size:         size,
		Leverage:     "3",
		ClientOID:    clientOID,
	}
	body, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("marshal intent: %v", err)
	}
	seq, err := store.AppendEvent(ctx, "trader", intent, body)
	if err != nil {
		t.Fatalf("append intent: %v", err)
	}

	if err := runtime.ProcessAvailable(ctx); err != nil {
		t.Fatalf("process available: %v", err)
	}
	executionEvents, err := store.ListEventsAfter(ctx, seq, 10, "execution")
	if err != nil {
		t.Fatalf("list execution events: %v", err)
	}
	if len(executionEvents) == 0 {
		t.Fatalf("expected execution reconcile event, got none")
	}
	if executionEvents[0].Kind != "execution.reconciled" {
		t.Fatalf("unexpected execution event: %+v", executionEvents[0])
	}
	openQty, err := waitForPositionQty(ctx, client, func(qty float64) bool { return qty > 0 })
	if err != nil {
		t.Fatalf("wait for open position: %v", err)
	}
	if openQty <= 0 {
		t.Fatalf("expected long position after execd runtime, got %f", openQty)
	}

	if err := runtime.FlattenSymbol(ctx, realSymbol); err != nil {
		t.Fatalf("flatten symbol: %v", err)
	}
	finalQty, err := waitForPositionQty(ctx, client, func(qty float64) bool { return math.Abs(qty) < 1e-9 })
	if err != nil {
		t.Fatalf("wait for flat position: %v", err)
	}
	if math.Abs(finalQty) > 1e-9 {
		t.Fatalf("expected final flat position, got %f", finalQty)
	}
}

func ensureFlat(ctx context.Context, runtime *Runtime, client *bitget.Client) error {
	qty, err := fetchPositionQty(ctx, client)
	if err != nil {
		return err
	}
	if math.Abs(qty) < 1e-9 {
		return nil
	}
	if err := runtime.FlattenSymbol(ctx, realSymbol); err != nil {
		return err
	}
	_, err = waitForPositionQty(ctx, client, func(qty float64) bool { return math.Abs(qty) < 1e-9 })
	return err
}

func waitForPositionQty(ctx context.Context, client *bitget.Client, ready func(float64) bool) (float64, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		qty, err := fetchPositionQty(ctx, client)
		if err == nil && ready(qty) {
			return qty, nil
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return 0, err
			}
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func fetchPositionQty(ctx context.Context, client *bitget.Client) (float64, error) {
	positions, err := client.FetchFuturesPositions(ctx, realProductType, realMarginCoin)
	if err != nil {
		return 0, err
	}
	var qty float64
	for _, position := range positions {
		if position.SymbolValue == realSymbol {
			qty += position.Qty
		}
	}
	return qty, nil
}
