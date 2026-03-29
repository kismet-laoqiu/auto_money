package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"quantlab/internal/market"
)

const (
	realTestSymbol      = "MSTRUSDT"
	realTestProductType = "USDT-FUTURES"
	realTestMarginCoin  = "USDT"
	realTestMinNotional = 5.0
)

func TestRealBitgetFetchFuturesCandles(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 to run real Bitget public tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := NewClient("")
	rules, err := client.FetchContractRules(ctx, "USDT-FUTURES")
	if err != nil {
		t.Fatalf("fetch contract rules: %v", err)
	}
	rule, ok := rules["MSTRUSDT"]
	if !ok {
		t.Fatalf("MSTRUSDT contract rule not found")
	}
	if rule.MinTradeNum != 0.01 || rule.MaxLeverage < 3 {
		t.Fatalf("unexpected MSTRUSDT rule: %+v", rule)
	}

	events, err := client.FetchCandles(ctx, "MSTRUSDT", "USDT-FUTURES", "1m", 5)
	if err != nil {
		t.Fatalf("fetch futures candles: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected non-empty candle response")
	}
	if events[0].SymbolValue != "MSTRUSDT" || events[0].Interval != "1m" {
		t.Fatalf("unexpected first candle: %+v", events[0])
	}
}

func TestRealBitgetPrivateReadSurface(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 to run real Bitget private tests")
	}
	creds := PrivateCredentials{
		Key:        os.Getenv("BITGET_API_KEY"),
		Secret:     os.Getenv("BITGET_API_SECRET"),
		Passphrase: os.Getenv("BITGET_PASSPHRASE"),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Skip("set BITGET_API_KEY BITGET_API_SECRET BITGET_PASSPHRASE to run real Bitget private tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := NewPrivateClient("", creds)
	accounts, err := client.FetchFuturesAccounts(ctx, "USDT-FUTURES")
	if err != nil {
		t.Fatalf("fetch futures accounts: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatalf("expected non-empty account response")
	}
	positions, err := client.FetchFuturesPositions(ctx, "USDT-FUTURES", "USDT")
	if err != nil {
		t.Fatalf("fetch futures positions: %v", err)
	}
	if positions == nil {
		t.Fatalf("expected non-nil position response")
	}
}

func TestRealBitgetEnsureFlatPosition(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" || os.Getenv("RUN_BITGET_CLEANUP") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 and RUN_BITGET_CLEANUP=1 to run real Bitget cleanup test")
	}
	creds := PrivateCredentials{
		Key:        os.Getenv("BITGET_API_KEY"),
		Secret:     os.Getenv("BITGET_API_SECRET"),
		Passphrase: os.Getenv("BITGET_PASSPHRASE"),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Skip("set BITGET_API_KEY BITGET_API_SECRET BITGET_PASSPHRASE to run real Bitget private tests")
	}
	client := NewPrivateClient("", creds)
	ensureFlatPosition(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	positions, err := client.FetchFuturesPositions(ctx, realTestProductType, realTestMarginCoin)
	if err != nil {
		t.Fatalf("fetch futures positions after cleanup: %v", err)
	}
	if qty := currentPositionQty(positions, realTestSymbol); math.Abs(qty) > 1e-9 {
		t.Fatalf("expected %s flat after cleanup, got qty=%f", realTestSymbol, qty)
	}
}

func TestBuildCloseRequestForHedgeModeLong(t *testing.T) {
	req := buildCloseRequest(0.04, "hedge_mode")
	if req.Side != "buy" || req.TradeSide != "close" || req.ReduceOnly != "" {
		t.Fatalf("unexpected hedge-mode close request: %+v", req)
	}
}

func TestBuildCloseRequestForOneWayModeLong(t *testing.T) {
	req := buildCloseRequest(0.04, "one_way_mode")
	if req.Side != "sell" || req.TradeSide != "" || req.ReduceOnly != "YES" {
		t.Fatalf("unexpected one-way close request: %+v", req)
	}
}

func TestComputeMinimumTradeSizeRoundsUpToValidStep(t *testing.T) {
	size := computeMinimumTradeSize(5, 99.2, 0.01, 0.01)
	if math.Abs(size-0.06) > 1e-9 {
		t.Fatalf("unexpected trade size: %f", size)
	}
}

func TestRealBitgetOrderLifecycleAndPrivateStream(t *testing.T) {
	if os.Getenv("RUN_BITGET_REAL") != "1" {
		t.Skip("set RUN_BITGET_REAL=1 to run real Bitget private tests")
	}
	creds := PrivateCredentials{
		Key:        os.Getenv("BITGET_API_KEY"),
		Secret:     os.Getenv("BITGET_API_SECRET"),
		Passphrase: os.Getenv("BITGET_PASSPHRASE"),
	}
	if creds.Key == "" || creds.Secret == "" || creds.Passphrase == "" {
		t.Skip("set BITGET_API_KEY BITGET_API_SECRET BITGET_PASSPHRASE to run real Bitget private tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	client := NewPrivateClient("", creds)
	orderSize, lastPrice, err := resolveLiveOrderSize(ctx)
	if err != nil {
		t.Fatalf("resolve live order size: %v", err)
	}
	t.Logf("real order size for %s on %s: size=%s last_price=%.4f min_notional=%.2f", realTestSymbol, time.Now().UTC().Format(time.RFC3339), orderSize, lastPrice, realTestMinNotional)
	initialPositions, err := client.FetchFuturesPositions(ctx, realTestProductType, realTestMarginCoin)
	if err != nil {
		t.Fatalf("fetch initial futures positions: %v", err)
	}
	if qty := currentPositionQty(initialPositions, realTestSymbol); math.Abs(qty) > 1e-9 {
		t.Skipf("%s must be flat before running real order lifecycle test, got qty=%f", realTestSymbol, qty)
	}
	defer ensureFlatPosition(t, client)

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	stream, err := startPrivateStreamRecorder(
		streamCtx,
		creds,
		PrivateSubscription{InstType: realTestProductType, Channel: "orders", InstID: realTestSymbol},
		PrivateSubscription{InstType: realTestProductType, Channel: "fill", InstID: realTestSymbol},
		PrivateSubscription{InstType: realTestProductType, Channel: "positions", InstID: "default"},
		PrivateSubscription{InstType: realTestProductType, Channel: "account", Coin: "default"},
	)
	if err != nil {
		t.Fatalf("start private stream recorder: %v", err)
	}
	ackCtx, ackCancel := context.WithTimeout(ctx, 15*time.Second)
	defer ackCancel()
	if err := stream.WaitForAck(ackCtx, "subscribe"); err != nil {
		t.Fatalf("wait for private subscribe ack: %v", err)
	}

	leverage, err := client.SetLeverage(ctx, SetLeverageRequest{
		Symbol:      realTestSymbol,
		ProductType: realTestProductType,
		MarginCoin:  realTestMarginCoin,
		Leverage:    "3",
		HoldSide:    "long",
	})
	if err != nil {
		t.Fatalf("set leverage: %v", err)
	}
	if leverage.MarginMode != "isolated" || leverage.LongLeverage != "3" {
		t.Fatalf("unexpected leverage setting: %+v", leverage)
	}

	openClientOID := fmt.Sprintf("ql-real-open-%d", time.Now().UTC().UnixMilli())
	openResult, err := client.PlaceOrder(ctx, PlaceOrderRequest{
		Symbol:      realTestSymbol,
		ProductType: realTestProductType,
		MarginMode:  "isolated",
		MarginCoin:  realTestMarginCoin,
		Side:        "buy",
		TradeSide:   "open",
		OrderType:   "market",
		Size:        orderSize,
		ClientOID:   openClientOID,
	})
	if err != nil {
		t.Fatalf("place open order: %v", err)
	}
	openDetail, err := waitForFilledOrder(ctx, client, OrderDetailRequest{
		Symbol:      realTestSymbol,
		ProductType: realTestProductType,
		OrderID:     openResult.OrderID,
		ClientOID:   openClientOID,
	})
	if err != nil {
		t.Fatalf("wait for open order detail: %v", err)
	}
	if openDetail.OrderID == "" || openDetail.Size <= 0 {
		t.Fatalf("unexpected open order detail: %+v", openDetail)
	}

	if _, err := stream.WaitForEvent(ctx, func(event market.MarketEvent) bool {
		order, ok := event.(market.OrderEvent)
		return ok && (order.ClientOID == openClientOID || order.OrderID == openResult.OrderID)
	}); err != nil {
		t.Fatalf("wait for private open order event: %v", err)
	}
	if _, err := stream.WaitForEvent(ctx, func(event market.MarketEvent) bool {
		position, ok := event.(market.PositionEvent)
		return ok && position.SymbolValue == realTestSymbol && position.Qty > 0
	}); err != nil {
		t.Fatalf("wait for private open position snapshot: %v", err)
	}
	if _, err := stream.WaitForEvent(ctx, func(event market.MarketEvent) bool {
		account, ok := event.(market.AccountEvent)
		return ok && account.MarginCoin == realTestMarginCoin
	}); err != nil {
		t.Fatalf("wait for private account snapshot: %v", err)
	}

	openQty, err := waitForPositionQty(ctx, client, func(qty float64) bool { return qty > 0 })
	if err != nil {
		t.Fatalf("wait for live open position: %v", err)
	}

	closeClientOID := fmt.Sprintf("ql-real-close-%d", time.Now().UTC().UnixMilli())
	positionSnapshot, err := fetchSinglePositionSnapshot(ctx, client)
	if err != nil {
		t.Fatalf("fetch single position before close: %v", err)
	}
	closeReq := buildCloseRequest(openQty, positionSnapshot.Mode)
	closeReq.ClientOID = closeClientOID
	closeResult, err := client.PlaceOrder(ctx, closeReq)
	if err != nil {
		t.Fatalf("place close order: %v", err)
	}
	closeDetail, err := waitForFilledOrder(ctx, client, OrderDetailRequest{
		Symbol:      realTestSymbol,
		ProductType: realTestProductType,
		OrderID:     closeResult.OrderID,
		ClientOID:   closeClientOID,
	})
	if err != nil {
		t.Fatalf("wait for close order detail: %v", err)
	}
	if closeDetail.OrderID == "" {
		t.Fatalf("unexpected close order detail: %+v", closeDetail)
	}
	if !strings.EqualFold(positionSnapshot.Mode, "hedge_mode") && !closeDetail.ReduceOnly {
		t.Fatalf("unexpected one-way close order detail: %+v", closeDetail)
	}

	if _, err := stream.WaitForEvent(ctx, func(event market.MarketEvent) bool {
		order, ok := event.(market.OrderEvent)
		return ok && (order.ClientOID == closeClientOID || order.OrderID == closeResult.OrderID)
	}); err != nil {
		t.Fatalf("wait for private close order event: %v", err)
	}
	finalQty, err := waitForPositionQty(ctx, client, func(qty float64) bool { return math.Abs(qty) < 1e-9 })
	if err != nil {
		t.Fatalf("wait for final flat position: %v", err)
	}
	if math.Abs(finalQty) > 1e-9 {
		t.Fatalf("expected final flat position, got %f", finalQty)
	}
}

type privateStreamRecorder struct {
	mu     sync.Mutex
	acks   []privateMessage
	events []market.MarketEvent
	err    error
}

func (recorder *privateStreamRecorder) setErr(err error) {
	if err == nil {
		return
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.err == nil {
		recorder.err = err
	}
}

func startPrivateStreamRecorder(ctx context.Context, creds PrivateCredentials, subscriptions ...PrivateSubscription) (*privateStreamRecorder, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, defaultPrivateWSURL, nil)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context canceled"), time.Now().Add(time.Second))
		_ = conn.Close()
	}()
	ts := fmt.Sprintf("%d", time.Now().UTC().Unix())
	if err := conn.WriteJSON(BuildLoginFrame(ts, creds, NewSigner(creds.Secret))); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := waitForPrivateEvent(ctx, conn, "login"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	rawCh := make(chan []byte, 16)
	recorder := capturePrivateStream(ctx, rawCh)
	args := make([]privateArg, 0, len(subscriptions))
	for _, sub := range subscriptions {
		args = append(args, privateArg{
			InstType: sub.InstType,
			Channel:  sub.Channel,
			InstID:   sub.InstID,
			Coin:     sub.Coin,
		})
	}
	go func() {
		defer close(rawCh)
		if err := conn.WriteJSON(privateSubscribeFrame{Op: "subscribe", Args: args}); err != nil {
			recorder.setErr(err)
			_ = conn.Close()
			return
		}
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				recorder.setErr(err)
				return
			}
			select {
			case <-ctx.Done():
				return
			case rawCh <- data:
			}
		}
	}()
	return recorder, nil
}

func capturePrivateStream(ctx context.Context, rawCh <-chan []byte) *privateStreamRecorder {
	recorder := &privateStreamRecorder{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				recorder.setErr(ctx.Err())
				return
			case raw, ok := <-rawCh:
				if !ok {
					if err := ctx.Err(); err != nil {
						recorder.setErr(err)
					} else {
						recorder.setErr(io.EOF)
					}
					return
				}
				var msg privateMessage
				if err := json.Unmarshal(raw, &msg); err == nil && msg.Event != "" {
					if strings.EqualFold(msg.Event, "error") || (msg.Code != "" && msg.Code != "0") {
						recorder.setErr(fmt.Errorf("private websocket event=%s code=%s msg=%s", msg.Event, msg.Code, msg.Msg))
						return
					}
					recorder.mu.Lock()
					recorder.acks = append(recorder.acks, msg)
					recorder.mu.Unlock()
					continue
				}
				events, err := DecodePrivateEvents(raw)
				if err != nil {
					recorder.setErr(err)
					return
				}
				recorder.mu.Lock()
				recorder.events = append(recorder.events, events...)
				recorder.mu.Unlock()
			}
		}
	}()
	return recorder
}

func (recorder *privateStreamRecorder) WaitForAck(ctx context.Context, event string) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		recorder.mu.Lock()
		for _, ack := range recorder.acks {
			if ack.Event == event {
				recorder.mu.Unlock()
				return nil
			}
		}
		err := recorder.err
		recorder.mu.Unlock()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (recorder *privateStreamRecorder) WaitForEvent(ctx context.Context, match func(market.MarketEvent) bool) (market.MarketEvent, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		recorder.mu.Lock()
		for _, event := range recorder.events {
			if match(event) {
				recorder.mu.Unlock()
				return event, nil
			}
		}
		err := recorder.err
		recorder.mu.Unlock()
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForFilledOrder(ctx context.Context, client *Client, req OrderDetailRequest) (OrderDetail, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		detail, err := client.GetOrderDetail(ctx, req)
		if err == nil && strings.Contains(strings.ToLower(detail.Status), "fill") {
			return detail, nil
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return OrderDetail{}, err
			}
			return OrderDetail{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForPositionQty(ctx context.Context, client *Client, ready func(float64) bool) (float64, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		positions, err := client.FetchFuturesPositions(ctx, realTestProductType, realTestMarginCoin)
		if err == nil {
			qty := currentPositionQty(positions, realTestSymbol)
			if ready(qty) {
				return qty, nil
			}
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func currentPositionQty(positions []market.PositionEvent, symbol string) float64 {
	var qty float64
	for _, position := range positions {
		if position.SymbolValue == symbol {
			qty += position.Qty
		}
	}
	return qty
}

func ensureFlatPosition(t *testing.T, client *Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	positions, err := client.FetchFuturesPositions(ctx, realTestProductType, realTestMarginCoin)
	if err != nil {
		t.Errorf("cleanup fetch futures positions: %v", err)
		return
	}
	qty := currentPositionQty(positions, realTestSymbol)
	if math.Abs(qty) < 1e-9 {
		return
	}

	snapshot, err := fetchSinglePositionSnapshot(ctx, client)
	if err != nil {
		t.Errorf("cleanup fetch single position: %v", err)
		return
	}
	closeReq := buildCloseRequest(snapshot.Qty, snapshot.Mode)
	closeReq.ClientOID = fmt.Sprintf("ql-cleanup-%d", time.Now().UTC().UnixMilli())
	if _, err := client.PlaceOrder(ctx, closeReq); err != nil {
		t.Errorf("cleanup close order: %v", err)
		return
	}
	if _, err := waitForPositionQty(ctx, client, func(qty float64) bool { return math.Abs(qty) < 1e-9 }); err != nil {
		t.Errorf("cleanup wait flat position: %v", err)
	}
}

type realPositionSnapshot struct {
	Qty  float64
	Mode string
}

func fetchSinglePositionSnapshot(ctx context.Context, client *Client) (realPositionSnapshot, error) {
	path := fmt.Sprintf(
		"/api/v2/mix/position/single-position?symbol=%s&productType=%s&marginCoin=%s",
		url.QueryEscape(realTestSymbol),
		url.QueryEscape(realTestProductType),
		url.QueryEscape(realTestMarginCoin),
	)
	body, err := client.doPrivate(ctx, http.MethodGet, path, nil)
	if err != nil {
		return realPositionSnapshot{}, err
	}
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			HoldSide string `json:"holdSide"`
			Total    string `json:"total"`
			PosMode  string `json:"posMode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return realPositionSnapshot{}, fmt.Errorf("decode single position response: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return realPositionSnapshot{}, fmt.Errorf("bitget single position code=%s msg=%s", response.Code, response.Msg)
	}
	if len(response.Data) == 0 {
		return realPositionSnapshot{}, fmt.Errorf("bitget single position returned empty data for %s", realTestSymbol)
	}
	qty, err := parseSignedPositionQty(response.Data[0].HoldSide, response.Data[0].Total)
	if err != nil {
		return realPositionSnapshot{}, err
	}
	return realPositionSnapshot{
		Qty:  qty,
		Mode: response.Data[0].PosMode,
	}, nil
}

func buildCloseRequest(qty float64, positionMode string) PlaceOrderRequest {
	req := PlaceOrderRequest{
		Symbol:      realTestSymbol,
		ProductType: realTestProductType,
		MarginMode:  "isolated",
		MarginCoin:  realTestMarginCoin,
		OrderType:   "market",
		Size:        strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", math.Abs(qty)), "0"), "."),
	}
	if strings.EqualFold(positionMode, "hedge_mode") {
		if qty < 0 {
			req.Side = "sell"
		} else {
			req.Side = "buy"
		}
		req.TradeSide = "close"
		return req
	}
	if qty < 0 {
		req.Side = "buy"
	} else {
		req.Side = "sell"
	}
	req.ReduceOnly = "YES"
	return req
}

func resolveLiveOrderSize(ctx context.Context) (string, float64, error) {
	client := NewClient("")
	rules, err := client.FetchContractRules(ctx, realTestProductType)
	if err != nil {
		return "", 0, err
	}
	rule, ok := rules[realTestSymbol]
	if !ok {
		return "", 0, fmt.Errorf("contract rule not found for %s", realTestSymbol)
	}
	lastPrice, err := client.FetchTickerPrice(ctx, realTestSymbol, realTestProductType)
	if err != nil || lastPrice <= 0 {
		candles, candleErr := client.FetchCandles(ctx, realTestSymbol, realTestProductType, "1m", 5)
		if candleErr != nil {
			if err != nil {
				return "", 0, fmt.Errorf("fetch ticker price: %v; fetch candles: %w", err, candleErr)
			}
			return "", 0, candleErr
		}
		if len(candles) == 0 {
			if err != nil {
				return "", 0, fmt.Errorf("fetch ticker price: %v; no candles returned for %s", err, realTestSymbol)
			}
			return "", 0, fmt.Errorf("no candles returned for %s", realTestSymbol)
		}
		lastPrice = candles[len(candles)-1].Close
	}
	size := computeMinimumTradeSize(realTestMinNotional, lastPrice, rule.MinTradeNum, rule.SizeMultiplier)
	return formatTradeSize(size), lastPrice, nil
}

func computeMinimumTradeSize(minNotional, lastPrice, minTradeNum, sizeMultiplier float64) float64 {
	if sizeMultiplier <= 0 {
		sizeMultiplier = minTradeNum
	}
	if sizeMultiplier <= 0 {
		sizeMultiplier = 0.01
	}
	size := minTradeNum
	if lastPrice > 0 {
		required := minNotional / lastPrice
		steps := math.Ceil(required / sizeMultiplier)
		if steps < 1 {
			steps = 1
		}
		size = steps * sizeMultiplier
	}
	if size < minTradeNum {
		size = minTradeNum
	}
	if lastPrice > 0 && size*lastPrice < minNotional {
		size += sizeMultiplier
	}
	return size
}

func formatTradeSize(size float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", size), "0"), ".")
}
