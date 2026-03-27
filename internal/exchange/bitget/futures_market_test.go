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
