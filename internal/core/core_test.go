package core

import "testing"

func TestWaveBias(t *testing.T) {
	longBias, shortBias := waveBias([]Pivot{{Price: 100, Kind: "low"}, {Price: 110, Kind: "high"}, {Price: 105, Kind: "low"}, {Price: 120, Kind: "high"}})
	if !longBias || shortBias {
		t.Fatalf("expected bullish wave bias, got long=%v short=%v", longBias, shortBias)
	}
}

func TestRetracementLevels(t *testing.T) {
	levels := retracementLevels(120, 100, true)
	if len(levels) != 3 {
		t.Fatalf("expected 3 fibonacci levels, got %d", len(levels))
	}
	if levels[0] >= 120 || levels[0] <= 100 {
		t.Fatalf("expected retracement level inside swing range, got %.2f", levels[0])
	}
}
