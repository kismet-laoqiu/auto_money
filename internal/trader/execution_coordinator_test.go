package trader

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildClientOIDStableAndShort(t *testing.T) {
	got := BuildClientOID("run-17", "BTCUSDT", 2, 1710000000000)
	if len(got) >= 50 {
		t.Fatalf("client oid too long: %s", got)
	}
	if got != BuildClientOID("run-17", "BTCUSDT", 2, 1710000000000) {
		t.Fatalf("client oid must be deterministic")
	}
}

func TestExitOrderAlwaysReduceOnly(t *testing.T) {
	req := BuildExitRequest(SymbolPosition{Symbol: "BTCUSDT", Qty: 0.02})
	if req.ReduceOnly != "YES" {
		t.Fatalf("exit request must be reduce-only: %+v", req)
	}
	if req.Side != "sell" || req.OrderType != "market" {
		t.Fatalf("unexpected exit request shape: %+v", req)
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal exit request: %v", err)
	}
	if !strings.Contains(string(body), "\"reduceOnly\":\"YES\"") {
		t.Fatalf("expected reduceOnly YES in json: %s", string(body))
	}
}
