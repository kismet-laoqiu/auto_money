package market

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AppendEventFunc func(ctx context.Context, source string, evt MarketEvent, raw []byte) (int64, error)

type BootstrapLoader interface {
	FetchCandles(ctx context.Context, symbol, productType, interval string, limit int) ([]BarClosedEvent, error)
}

type PrivateBootstrapLoader interface {
	FetchFuturesAccounts(ctx context.Context, productType string) ([]AccountEvent, error)
	FetchFuturesPositions(ctx context.Context, productType, marginCoin string) ([]PositionEvent, error)
}

type RuntimeConfig struct {
	Symbol         string
	ProductType    string
	MarginCoin     string
	Interval       string
	BootstrapLimit int
	AppendEvent    AppendEventFunc
	Loader         BootstrapLoader
	PrivateLoader  PrivateBootstrapLoader
	Source         EventSource
	Decoder        PublicFeed
	PrivateSource  EventSource
	PrivateDecoder PrivateFeed
	Aggregator     *MicroBarAggregator
}

type Runtime struct {
	cfg RuntimeConfig
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.Interval == "" {
		cfg.Interval = "1m"
	}
	if cfg.BootstrapLimit <= 0 {
		cfg.BootstrapLimit = 10
	}
	return &Runtime{cfg: cfg}
}

func (runtime *Runtime) Run(ctx context.Context) error {
	if runtime.cfg.AppendEvent == nil {
		return fmt.Errorf("market runtime append_event is nil")
	}
	if runtime.cfg.Source != nil && runtime.cfg.Decoder == nil {
		return fmt.Errorf("market runtime decoder is nil")
	}
	if runtime.cfg.PrivateSource != nil && runtime.cfg.PrivateDecoder == nil {
		return fmt.Errorf("market runtime private_decoder is nil")
	}
	if err := runtime.bootstrap(ctx); err != nil {
		return err
	}
	consumers := 0
	errCh := make(chan error, 2)
	if runtime.cfg.Source != nil {
		consumers++
		go func() {
			errCh <- runtime.runPublic(ctx)
		}()
	}
	if runtime.cfg.PrivateSource != nil {
		consumers++
		go func() {
			errCh <- runtime.runPrivate(ctx)
		}()
	}
	for i := 0; i < consumers; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) bootstrap(ctx context.Context) error {
	if runtime.cfg.Loader != nil {
		events, err := runtime.cfg.Loader.FetchCandles(ctx, runtime.cfg.Symbol, runtime.cfg.ProductType, runtime.cfg.Interval, runtime.cfg.BootstrapLimit)
		if err != nil {
			return err
		}
		events = filterClosedBootstrapBars(events, runtime.cfg.Interval, time.Now().UTC())
		for _, event := range events {
			if err := runtime.appendEvent(ctx, "market.bootstrap", event); err != nil {
				return err
			}
		}
	}
	if runtime.cfg.PrivateLoader == nil {
		return nil
	}
	accounts, err := runtime.cfg.PrivateLoader.FetchFuturesAccounts(ctx, runtime.cfg.ProductType)
	if err != nil {
		return err
	}
	for _, event := range accounts {
		if err := runtime.appendEvent(ctx, "market.private", event); err != nil {
			return err
		}
	}
	positions, err := runtime.cfg.PrivateLoader.FetchFuturesPositions(ctx, runtime.cfg.ProductType, runtime.cfg.MarginCoin)
	if err != nil {
		return err
	}
	for _, event := range positions {
		if err := runtime.appendEvent(ctx, "market.private", event); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) runPublic(ctx context.Context) error {
	feed := NewPublicFeed(runtime.cfg.Source, runtime.cfg.Decoder, runtime.cfg.Aggregator)
	for event := range feed.Events(ctx) {
		if err := runtime.appendEvent(ctx, "market.public", event); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (runtime *Runtime) runPrivate(ctx context.Context) error {
	rawCh := runtime.cfg.PrivateSource.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case raw, ok := <-rawCh:
			if !ok {
				return ctx.Err()
			}
			events, err := runtime.cfg.PrivateDecoder.Decode(raw)
			if err != nil {
				continue
			}
			for _, event := range events {
				if err := runtime.appendEvent(ctx, "market.private", event); err != nil {
					return err
				}
			}
		}
	}
}

func (runtime *Runtime) appendEvent(ctx context.Context, source string, event MarketEvent) error {
	_, err := runtime.cfg.AppendEvent(ctx, source, event, mustJSON(event))
	if isDuplicateEventErr(err) {
		return nil
	}
	return err
}

func isDuplicateEventErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "event_log.event_id")
}

func mustJSON(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return body
}

func filterClosedBootstrapBars(events []BarClosedEvent, interval string, now time.Time) []BarClosedEvent {
	step, ok := bootstrapIntervalDuration(interval)
	if !ok {
		return append([]BarClosedEvent(nil), events...)
	}
	out := make([]BarClosedEvent, 0, len(events))
	for _, event := range events {
		if event.Ts.UTC().Add(step).After(now.UTC()) {
			continue
		}
		out = append(out, event)
	}
	return out
}

func bootstrapIntervalDuration(interval string) (time.Duration, bool) {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "1m":
		return time.Minute, true
	case "5m":
		return 5 * time.Minute, true
	case "15m":
		return 15 * time.Minute, true
	case "1h":
		return time.Hour, true
	case "4h":
		return 4 * time.Hour, true
	case "1d":
		return 24 * time.Hour, true
	case "1w":
		return 7 * 24 * time.Hour, true
	default:
		return 0, false
	}
}
