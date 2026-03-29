package market_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quantlab/internal/exchange/bitget"
	"quantlab/internal/market"
	sqlitepkg "quantlab/internal/store/sqlite"
)

func TestRealBitgetPublicRuntimePersistsBootstrapAndPublicEvents(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 to run real Bitget public tests")
	}
	store, err := sqlitepkg.NewStore(filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	runtime := market.NewRuntime(market.RuntimeConfig{
		Symbol:         "MSTRUSDT",
		ProductType:    "USDT-FUTURES",
		Interval:       "1m",
		BootstrapLimit: 5,
		AppendEvent: func(ctx context.Context, source string, evt market.MarketEvent, raw []byte) (int64, error) {
			return store.AppendEvent(ctx, source, evt, raw)
		},
		Loader: bitget.NewClient(""),
		Source: bitget.NewPublicWSSource(
			"",
			bitget.PublicSubscription{InstType: "USDT-FUTURES", Channel: "trade", InstID: "MSTRUSDT"},
			bitget.PublicSubscription{InstType: "USDT-FUTURES", Channel: "candle1m", InstID: "MSTRUSDT"},
		),
		Decoder: bitget.NewPublicWSDecoder(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- runtime.Run(ctx)
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var foundBootstrap bool
	var foundTrade bool
	var foundBar bool
	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("runtime exited with error: %v", err)
			}
			if !(foundBootstrap && foundTrade && foundBar) {
				t.Fatalf("runtime exited before collecting required events: bootstrap=%t trade=%t bar=%t", foundBootstrap, foundTrade, foundBar)
			}
			return
		case <-ticker.C:
			events, err := store.ListEventsAfter(context.Background(), 0, 512)
			if err != nil {
				t.Fatalf("list events: %v", err)
			}
			for _, event := range events {
				if event.Source == "market.bootstrap" && event.Kind == "bar_closed" {
					foundBootstrap = true
				}
				if event.Source == "market.public" && event.Kind == "trade_tick" {
					foundTrade = true
				}
				if event.Source == "market.public" && event.Kind == "bar_closed" {
					foundBar = true
				}
			}
			if foundBootstrap && foundTrade && foundBar {
				cancel()
				err := <-errCh
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Fatalf("runtime exited with error: %v", err)
				}
				return
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for required public events: bootstrap=%t trade=%t bar=%t", foundBootstrap, foundTrade, foundBar)
		}
	}
}

func TestRealBitgetPrivateBootstrapPersistsSnapshots(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 to run real Bitget private tests")
	}
	creds := bitget.PrivateCredentials{
		Key:        os.Getenv("BITGET_API_KEY"),
		Secret:     os.Getenv("BITGET_API_SECRET"),
		Passphrase: os.Getenv("BITGET_PASSPHRASE"),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Skip("set BITGET_API_KEY BITGET_API_SECRET BITGET_PASSPHRASE to run real Bitget private tests")
	}
	store, err := sqlitepkg.NewStore(filepath.Join(t.TempDir(), "runtime-private-bootstrap.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	runtime := market.NewRuntime(market.RuntimeConfig{
		ProductType:   "USDT-FUTURES",
		MarginCoin:    "USDT",
		AppendEvent:   func(ctx context.Context, source string, evt market.MarketEvent, raw []byte) (int64, error) { return store.AppendEvent(ctx, source, evt, raw) },
		PrivateLoader: bitget.NewPrivateClient("", creds),
	})
	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("run private bootstrap runtime: %v", err)
	}
	events, err := store.ListEventsAfter(context.Background(), 0, 64)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	var foundAccount bool
	for _, event := range events {
		if event.Source == "market.private" && event.Kind == "account_snapshot" {
			foundAccount = true
		}
	}
	if !foundAccount {
		t.Fatalf("expected account snapshot in market.private, got %+v", events)
	}
}

func TestRealBitgetPrivateStreamPersistsSnapshots(t *testing.T) {
	t.Skip("Bitget contract private channels do not push snapshots on first subscription; verify private websocket via the real order e2e flow instead.")
}
