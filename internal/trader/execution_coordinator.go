package trader

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"quantlab/internal/core"
	"quantlab/internal/exchange/bitget"
)

const (
	defaultEntryQtyValue = 0.01
	defaultEntryQtyText  = "0.01"
)

type LiveExchange interface {
	PlaceOrder(req bitget.PlaceOrderRequest) error
}

type SymbolPosition struct {
	Symbol      string
	Qty         float64
	ProductType string
	MarginMode  string
	MarginCoin  string
}

func BuildClientOID(runID, symbol string, tranche int, tsMillis int64) string {
	short := strings.ToLower(strings.TrimSuffix(symbol, "USDT"))
	return fmt.Sprintf("ql-%s-%s-%d-%d", runID, short, tranche, tsMillis)
}

func BuildEntryRequest(runID string, candidate Candidate) bitget.PlaceOrderRequest {
	side := "buy"
	if candidate.Side == core.Short {
		side = "sell"
	}
	return bitget.PlaceOrderRequest{
		Symbol:      candidate.Symbol,
		ProductType: "USDT-FUTURES",
		MarginMode:  "isolated",
		MarginCoin:  "USDT",
		Side:        side,
		TradeSide:   "open",
		OrderType:   "market",
		Size:        defaultEntryQtyText,
		ClientOID:   BuildClientOID(defaultString(runID, "runtime"), candidate.Symbol, 1, candidate.Ts.UnixMilli()),
	}
}

func BuildExitRequest(position SymbolPosition) bitget.PlaceOrderRequest {
	side := "sell"
	if position.Qty < 0 {
		side = "buy"
	}
	return bitget.PlaceOrderRequest{
		Symbol:      position.Symbol,
		ProductType: defaultString(position.ProductType, "USDT-FUTURES"),
		MarginMode:  defaultString(position.MarginMode, "isolated"),
		MarginCoin:  defaultString(position.MarginCoin, "USDT"),
		Side:        side,
		OrderType:   "market",
		Size:        strconv.FormatFloat(math.Abs(position.Qty), 'f', -1, 64),
		ReduceOnly:  "YES",
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
