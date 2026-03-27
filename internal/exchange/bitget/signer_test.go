package bitget

import "testing"

func TestSignRequestDeterministic(t *testing.T) {
	signer := NewSigner("secret123")
	got := signer.Sign("1700000000000", "GET", "/api/v2/mix/market/contracts?productType=USDT-FUTURES", "")
	want := "I2oPv1u2vVUHRuN0JVujGq5Iccgl0a7/v74shxuv2XU="
	if got != want {
		t.Fatalf("unexpected signature: %s", got)
	}
}
