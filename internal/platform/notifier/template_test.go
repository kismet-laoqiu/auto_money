package notifier

import "testing"

func TestNotificationRenderSnapshots(t *testing.T) {
	cases := []struct {
		name         string
		notification Notification
		want         string
	}{
		{
			name: "promotion approved",
			notification: Notification{
				Kind:    KindPromotionApproved,
				Title:   "mstr-wave-fib v0.1.0 promotion approved",
				Summary: "promotion is live_active",
				Details: []string{"promotion_id=promo-1", "strategy=mstr-wave-fib", "version=v0.1.0"},
			},
			want: "上线审批通过\nmstr-wave-fib v0.1.0 promotion approved\npromotion is live_active\npromotion_id=promo-1\nstrategy=mstr-wave-fib\nversion=v0.1.0",
		},
		{
			name: "canary degraded",
			notification: Notification{
				Kind:    KindCanaryDegraded,
				Title:   "mstr-wave-fib v0.1.0 canary degraded",
				Summary: "canary gate degraded",
				Details: []string{"reason=shadow/live mismatch"},
			},
			want: "canary 降级\nmstr-wave-fib v0.1.0 canary degraded\ncanary gate degraded\nreason=shadow/live mismatch",
		},
		{
			name: "daily summary",
			notification: Notification{
				Kind:    KindDailySummary,
				Title:   "每日 summary",
				Summary: "MSTRUSDT fill avgPrice=126.33",
				Details: []string{"strategy=mstr-wave-fib", "status=live"},
			},
			want: "每日总结\n每日 summary\nMSTRUSDT fill avgPrice=126.33\nstrategy=mstr-wave-fib\nstatus=live",
		},
		{
			name: "daily signal",
			notification: Notification{
				Kind:    KindDailySignal,
				Title:   "BTCUSDT 日线买点",
				Summary: "score=4.82 threshold=4.60 close=81234.1",
				Details: []string{"reasons=trend up, break retest"},
			},
			want: "日线信号\nBTCUSDT 日线买点\nscore=4.82 threshold=4.60 close=81234.1\nreasons=trend up, break retest",
		},
		{
			name: "market alert",
			notification: Notification{
				Kind:    KindMarketAlert,
				Title:   "BTCUSDT 15m 暴涨",
				Summary: "move_pct=4.81 threshold=3.25",
				Details: []string{"volume_ratio=4.20 volume_zscore=5.12"},
			},
			want: "市场异动\nBTCUSDT 15m 暴涨\nmove_pct=4.81 threshold=3.25\nvolume_ratio=4.20 volume_zscore=5.12",
		},
	}
	for _, tc := range cases {
		if got := tc.notification.Render(); got != tc.want {
			t.Fatalf("%s: unexpected render\nwant:\n%s\n\ngot:\n%s", tc.name, tc.want, got)
		}
	}
}
