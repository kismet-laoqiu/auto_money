package notifier

import (
	"testing"
	"time"
)

func TestDefaultDailySummaryScheduleUsesAsiaShanghai(t *testing.T) {
	schedule := DefaultDailySummarySchedule()
	if schedule.LocationName != "Asia/Shanghai" {
		t.Fatalf("unexpected location: %+v", schedule)
	}
	next, err := schedule.NextAfter(time.Date(2026, 3, 29, 11, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next after: %v", err)
	}
	if !next.Equal(time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected same-day next run: %s", next)
	}
}

func TestDailySummaryScheduleRollsToNextShanghaiDay(t *testing.T) {
	next, err := DefaultDailySummarySchedule().NextAfter(time.Date(2026, 3, 29, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next after: %v", err)
	}
	if !next.Equal(time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected next-day run: %s", next)
	}
}
