package trader

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"quantlab/internal/core"
)

const (
	intentMinNotionalUSDT = 5.0
	intentQtyStep         = 0.01
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

func ComputeIntentSizeText(entryPrice float64) string {
	price := entryPrice
	if price <= 0 {
		price = 1
	}
	steps := math.Ceil((intentMinNotionalUSDT / price) / intentQtyStep)
	if steps < 1 {
		steps = 1
	}
	return strconv.FormatFloat(steps*intentQtyStep, 'f', 2, 64)
}

func parseIntentSize(size string) float64 {
	qty, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return 0
	}
	return qty
}

func signedIntentQty(intent EntryIntentEvent) float64 {
	qty := parseIntentSize(intent.Size)
	if intent.Side == core.Short {
		return -qty
	}
	return qty
}
