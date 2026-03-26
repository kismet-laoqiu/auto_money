package adapters

import (
	"strings"
	"testing"
	"time"

	"quantlab/internal/core"
)

func TestParseBitgetStreamBar(t *testing.T) {
	bar, err := parseBitgetStreamBar([]string{"1710000000000", "100", "105", "95", "102", "321.5"})
	if err != nil {
		t.Fatalf("parse stream bar: %v", err)
	}
	if bar.Time.UTC().Format(time.RFC3339) != "2024-03-09T16:00:00Z" {
		t.Fatalf("unexpected time: %s", bar.Time.UTC().Format(time.RFC3339))
	}
	if bar.Close != 102 || bar.Volume != 321.5 {
		t.Fatalf("unexpected bar values: %+v", bar)
	}
}

func TestUpdateStreamCacheAdvancesOnlyOnNewBar(t *testing.T) {
	start := time.Date(2024, 3, 9, 16, 0, 0, 0, time.UTC)
	cache := &streamCache{Bars: []core.Bar{{Time: start, Close: 100}}}

	closedIndex, advanced := updateStreamCache(cache, core.Bar{Time: start, Close: 101})
	if advanced || closedIndex != -1 {
		t.Fatalf("expected in-place update, got advanced=%v closedIndex=%d", advanced, closedIndex)
	}
	if cache.Bars[0].Close != 101 {
		t.Fatalf("expected last bar replacement, got %.2f", cache.Bars[0].Close)
	}

	closedIndex, advanced = updateStreamCache(cache, core.Bar{Time: start.Add(time.Minute), Close: 102})
	if !advanced || closedIndex != 0 {
		t.Fatalf("expected closed previous bar, got advanced=%v closedIndex=%d", advanced, closedIndex)
	}
	if len(cache.Bars) != 2 {
		t.Fatalf("expected appended bar, got %d", len(cache.Bars))
	}
}

func TestBitgetCandleChannel(t *testing.T) {
	if channel := bitgetCandleChannel("1h"); channel != "candle1H" {
		t.Fatalf("unexpected 1h channel: %s", channel)
	}
	if channel := bitgetCandleChannel("unknown"); channel != "candle1m" {
		t.Fatalf("unexpected fallback channel: %s", channel)
	}
}

func TestFormatStreamAlert(t *testing.T) {
	alert := formatStreamAlert("BTCUSDT", "1m", core.Bar{Time: time.Date(2024, 3, 9, 16, 0, 0, 0, time.UTC), Close: 102.5}, core.Signal{
		Side:    core.Long,
		Score:   4.5,
		Entry:   102.5,
		Stop:    100,
		Target:  107.5,
		Reasons: []string{"SMA trend up", "support reaction"},
	})
	if !strings.Contains(alert, "symbol=BTCUSDT") || !strings.Contains(alert, "reasons=SMA trend up, support reaction") {
		t.Fatalf("unexpected alert body: %s", alert)
	}
}
