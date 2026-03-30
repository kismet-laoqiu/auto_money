package notifier

import (
	"context"
	"testing"
	"time"

	sqlitepkg "quantlab/internal/store/sqlite"
	"quantlab/internal/trader"
)

func TestRuntimeHealthReportsCursorAndSchedule(t *testing.T) {
	store := &runtimeStoreStub{
		loadedCursor: 5,
		events: []sqlitepkg.EventEnvelope{
			{
				Seq:        11,
				Source:     "trader",
				EventID:    "risk-1",
				Symbol:     "MSTRUSDT",
				Kind:       "risk.state_changed",
				ExchangeTS: time.Unix(1710000000, 0).UTC(),
				Payload: mustJSON(t, trader.RiskEvent{
					EventIDValue: "risk-1",
					SymbolValue:  "MSTRUSDT",
					Ts:           time.Unix(1710000000, 0).UTC(),
					From:         trader.ArmingArmed,
					To:           trader.ArmingHalted,
					Reason:       "health check",
				}),
			},
		},
	}
	runtime := NewRuntime(RuntimeConfig{Store: store, Service: &sinkStub{}})
	initial := runtime.Health()
	if initial.ConsumerKey != "notifierd" || initial.Cursor != 0 {
		t.Fatalf("unexpected initial health: %+v", initial)
	}
	if initial.DailySummary.LocationName != "Asia/Shanghai" {
		t.Fatalf("unexpected default daily summary config: %+v", initial)
	}

	if err := runtime.ProcessAvailable(context.Background()); err != nil {
		t.Fatalf("process available: %v", err)
	}
	health := runtime.Health()
	if !health.Loaded || health.Cursor != 11 || health.MaxAttempts != 3 {
		t.Fatalf("unexpected runtime health after process: %+v", health)
	}
}
