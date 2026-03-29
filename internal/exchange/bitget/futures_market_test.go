package bitget

import "testing"

func TestDecodeContractsResponse(t *testing.T) {
	payload := []byte(`{"code":"00000","data":[{"symbol":"BTCUSDT","minTradeNum":"0.001","sizeMultiplier":"0.001","maxLeverage":"125"}]}`)
	rules, err := decodeContractRules(payload)
	if err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	if rules["BTCUSDT"].MinTradeNum != 0.001 {
		t.Fatalf("unexpected min trade num: %+v", rules["BTCUSDT"])
	}
}

func TestDecodeFuturesCandlesResponse(t *testing.T) {
	payload := []byte(`{"code":"00000","data":[["1710000000000","62000","62100","61950","62050","12.5"],["1710000060000","62050","62200","62000","62180","8.5"]]}`)
	events, err := decodeCandleEvents("MSTRUSDT", "1m", payload)
	if err != nil {
		t.Fatalf("decode candle events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[0].SymbolValue != "MSTRUSDT" || events[0].Interval != "1m" || events[0].Close != 62050 {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[1].High != 62200 || events[1].Volume != 8.5 {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
}

func TestDecodeFuturesTickerResponse(t *testing.T) {
	payload := []byte(`{"code":"00000","data":[{"symbol":"MSTRUSDT","lastPr":"126.19","markPrice":"126.18"}]}`)
	price, err := decodeTickerPrice(payload)
	if err != nil {
		t.Fatalf("decode ticker price: %v", err)
	}
	if price != 126.19 {
		t.Fatalf("unexpected ticker price: %f", price)
	}
}

func TestDecodeFuturesTickerResponseFallsBackToMarkPrice(t *testing.T) {
	payload := []byte(`{"code":"00000","data":[{"symbol":"MSTRUSDT","markPrice":"126.18"}]}`)
	price, err := decodeTickerPrice(payload)
	if err != nil {
		t.Fatalf("decode ticker price with mark fallback: %v", err)
	}
	if price != 126.18 {
		t.Fatalf("unexpected fallback ticker price: %f", price)
	}
}
