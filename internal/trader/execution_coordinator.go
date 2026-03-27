package trader

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"quantlab/internal/exchange/bitget"
)

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
