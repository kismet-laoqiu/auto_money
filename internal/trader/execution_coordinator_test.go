package trader

import "testing"

func TestBuildClientOIDStableAndShort(t *testing.T) {
	got := BuildClientOID("run-17", "BTCUSDT", 2, 1710000000000)
	if len(got) >= 50 {
		t.Fatalf("client oid too long: %s", got)
	}
	if got != BuildClientOID("run-17", "BTCUSDT", 2, 1710000000000) {
		t.Fatalf("client oid must be deterministic")
	}
}

func TestComputeIntentSizeTextRoundsUpToMinNotional(t *testing.T) {
	if got := ComputeIntentSizeText(125); got != "0.04" {
		t.Fatalf("expected 0.04 at price 125, got %s", got)
	}
	if got := ComputeIntentSizeText(103); got != "0.05" {
		t.Fatalf("expected 0.05 at price 103, got %s", got)
	}
}
