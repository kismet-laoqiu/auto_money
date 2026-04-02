package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
	sqlitepkg "quantlab/internal/store/sqlite"
)

type RuntimeStore interface {
	AppendEvent(ctx context.Context, source string, evt sqlitepkg.LogEvent, raw []byte) (int64, error)
	ListEventsAfter(ctx context.Context, afterSeq int64, limit int, sources ...string) ([]sqlitepkg.EventEnvelope, error)
	SaveConsumerCursor(ctx context.Context, consumer string, seq int64) error
	LoadConsumerCursor(ctx context.Context, consumer string) (int64, error)
}

type ExchangeClient interface {
	SetLeverage(ctx context.Context, req bitget.SetLeverageRequest) (bitget.LeverageSetting, error)
	PlaceOrder(ctx context.Context, req bitget.PlaceOrderRequest) (bitget.OrderResult, error)
	GetOrderDetail(ctx context.Context, req bitget.OrderDetailRequest) (bitget.OrderDetail, error)
	FetchSinglePosition(ctx context.Context, symbol, productType, marginCoin string) (bitget.SinglePositionSnapshot, error)
}

type RuntimeConfig struct {
	ConsumerKey       string
	OutputSource      string
	Sources           []string
	BatchSize         int
	PollInterval      time.Duration
	OrderPollInterval time.Duration
	ProductType       string
	MarginMode        string
	MarginCoin        string
	Store             RuntimeStore
	Exchange          ExchangeClient
}

type Runtime struct {
	cfg    RuntimeConfig
	cursor int64
	loaded bool
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.ConsumerKey == "" {
		cfg.ConsumerKey = "execd"
	}
	if cfg.OutputSource == "" {
		cfg.OutputSource = "execution"
	}
	if len(cfg.Sources) == 0 {
		cfg.Sources = []string{"trader"}
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 128
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.OrderPollInterval <= 0 {
		cfg.OrderPollInterval = time.Second
	}
	if cfg.ProductType == "" {
		cfg.ProductType = "USDT-FUTURES"
	}
	if cfg.MarginMode == "" {
		cfg.MarginMode = "isolated"
	}
	if cfg.MarginCoin == "" {
		cfg.MarginCoin = "USDT"
	}
	return &Runtime{cfg: cfg}
}

func (runtime *Runtime) Run(ctx context.Context) error {
	ticker := time.NewTicker(runtime.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) ProcessAvailable(ctx context.Context) error {
	if err := runtime.ensureLoaded(ctx); err != nil {
		return err
	}
	events, err := runtime.cfg.Store.ListEventsAfter(ctx, runtime.cursor, runtime.cfg.BatchSize, runtime.cfg.Sources...)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := runtime.handleEnvelope(ctx, event); err != nil {
			return err
		}
		runtime.cursor = event.Seq
		if err := runtime.cfg.Store.SaveConsumerCursor(ctx, runtime.cfg.ConsumerKey, runtime.cursor); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) FlattenSymbol(ctx context.Context, symbol string) error {
	if runtime.cfg.Exchange == nil {
		return fmt.Errorf("execution exchange is nil")
	}
	position, err := runtime.cfg.Exchange.FetchSinglePosition(ctx, symbol, runtime.cfg.ProductType, runtime.cfg.MarginCoin)
	if err != nil {
		return err
	}
	if math.Abs(position.Qty) < 1e-9 {
		return nil
	}
	req := buildCloseRequest(symbol, runtime.cfg.ProductType, runtime.cfg.MarginMode, runtime.cfg.MarginCoin, position)
	result, err := runtime.cfg.Exchange.PlaceOrder(ctx, req)
	if err != nil {
		return err
	}
	_, err = runtime.waitForFilledOrder(ctx, bitget.OrderDetailRequest{
		Symbol:      symbol,
		ProductType: runtime.cfg.ProductType,
		OrderID:     result.OrderID,
		ClientOID:   firstNonEmpty(result.ClientOID, req.ClientOID),
	})
	return err
}

func (runtime *Runtime) ensureLoaded(ctx context.Context) error {
	if runtime.loaded {
		return nil
	}
	if runtime.cfg.Store == nil {
		return fmt.Errorf("execution runtime store is nil")
	}
	cursor, err := runtime.cfg.Store.LoadConsumerCursor(ctx, runtime.cfg.ConsumerKey)
	if err != nil {
		return err
	}
	runtime.cursor = cursor
	runtime.loaded = true
	return nil
}

func (runtime *Runtime) handleEnvelope(ctx context.Context, event sqlitepkg.EventEnvelope) error {
	if event.Kind != "entry.intent.created" {
		return nil
	}
	intent, err := DecodeEntryIntentEvent(event.Payload)
	if err != nil {
		return err
	}
	return runtime.executeIntent(ctx, intent)
}

func (runtime *Runtime) executeIntent(ctx context.Context, intent EntryIntentEvent) error {
	if runtime.cfg.Exchange == nil {
		return fmt.Errorf("execution exchange is nil")
	}
	executionVenue := strings.ToLower(strings.TrimSpace(intent.ExecutionVenue))
	if executionVenue == "" {
		executionVenue = "bitget"
	}
	if executionVenue != "bitget" {
		return fmt.Errorf("unsupported execution venue: %s", executionVenue)
	}
	productType := firstNonEmpty(intent.ProductType, runtime.cfg.ProductType)
	marginMode := firstNonEmpty(intent.MarginMode, runtime.cfg.MarginMode)
	marginCoin := firstNonEmpty(intent.MarginCoin, runtime.cfg.MarginCoin)
	leverage := firstNonEmpty(intent.Leverage, "3")
	holdSide := holdSideForSide(intent.Side)
	if _, err := runtime.cfg.Exchange.SetLeverage(ctx, bitget.SetLeverageRequest{
		Symbol:      intent.SymbolValue,
		ProductType: productType,
		MarginCoin:  marginCoin,
		Leverage:    leverage,
		HoldSide:    holdSide,
	}); err != nil {
		return err
	}
	orderResult, err := runtime.cfg.Exchange.PlaceOrder(ctx, bitget.PlaceOrderRequest{
		Symbol:      intent.SymbolValue,
		ProductType: productType,
		MarginMode:  marginMode,
		MarginCoin:  marginCoin,
		Side:        openSideForSignal(intent.Side),
		TradeSide:   "open",
		OrderType:   "market",
		Size:        intent.Size,
		ClientOID:   intent.ClientOID,
	})
	if err != nil {
		return err
	}
	detail, err := runtime.waitForFilledOrder(ctx, bitget.OrderDetailRequest{
		Symbol:      intent.SymbolValue,
		ProductType: productType,
		OrderID:     orderResult.OrderID,
		ClientOID:   firstNonEmpty(orderResult.ClientOID, intent.ClientOID),
	})
	if err != nil {
		return err
	}
	return runtime.appendEvent(ctx, BuildReconcileEvent(intent, detail))
}

func (runtime *Runtime) waitForFilledOrder(ctx context.Context, req bitget.OrderDetailRequest) (bitget.OrderDetail, error) {
	ticker := time.NewTicker(runtime.cfg.OrderPollInterval)
	defer ticker.Stop()
	for {
		detail, err := runtime.cfg.Exchange.GetOrderDetail(ctx, req)
		if err == nil && strings.Contains(strings.ToLower(detail.Status), "fill") {
			return detail, nil
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return bitget.OrderDetail{}, err
			}
			return bitget.OrderDetail{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) appendEvent(ctx context.Context, event sqlitepkg.LogEvent) error {
	if runtime.cfg.Store == nil {
		return nil
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = runtime.cfg.Store.AppendEvent(ctx, runtime.cfg.OutputSource, event, body)
	return err
}

func buildCloseRequest(symbol, productType, marginMode, marginCoin string, position bitget.SinglePositionSnapshot) bitget.PlaceOrderRequest {
	req := bitget.PlaceOrderRequest{
		Symbol:      symbol,
		ProductType: productType,
		MarginMode:  marginMode,
		MarginCoin:  marginCoin,
		OrderType:   "market",
		Size:        formatTradeSize(math.Abs(position.Qty)),
	}
	if strings.EqualFold(position.Mode, "hedge_mode") {
		if position.Qty < 0 {
			req.Side = "sell"
		} else {
			req.Side = "buy"
		}
		req.TradeSide = "close"
		return req
	}
	if position.Qty < 0 {
		req.Side = "buy"
	} else {
		req.Side = "sell"
	}
	req.ReduceOnly = "YES"
	return req
}

func holdSideForSide(side core.Side) string {
	if side == core.Short {
		return "short"
	}
	return "long"
}

func openSideForSignal(side core.Side) string {
	if side == core.Short {
		return "sell"
	}
	return "buy"
}

func formatTradeSize(size float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(size, 'f', 8, 64), "0"), ".")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
