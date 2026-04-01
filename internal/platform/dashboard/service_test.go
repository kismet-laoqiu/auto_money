package dashboard

import (
	"context"
	"strings"
	"testing"
	"time"

	"quantlab/internal/platform/insights"
)

func TestServiceRenderHTMLIncludesMarkdownAndForms(t *testing.T) {
	service := NewService(Config{
		Insights: reporterStub{
			report: insights.DashboardReport{
				GeneratedAt: time.Unix(1710000000, 0).UTC(),
				Symbols: []insights.SymbolSnapshot{{
					Symbol:      "BTCUSDT",
					LatestPrice: 81234.5,
					DailyFeatures: insights.FeatureSnapshot{
						RSI14:        31.2,
						VolumeZScore: 1.8,
						TrendUp:      true,
					},
				}},
				MarketContext: &insights.MarketContextSnapshot{
					BTC: &insights.MarketAssetSnapshot{
						Symbol:        "BTCUSDT",
						CurrentPrice:  81234.5,
						WMA200:        59436,
						PriceToWMA200: 1.37,
					},
					FearGreed: &insights.FearGreedSnapshot{
						Value:          11,
						Classification: "Extreme Fear",
					},
				},
			},
		},
	})

	html, err := service.RenderHTML(context.Background(), "BTCUSDT\nETHUSDT", "saved")
	if err != nil {
		t.Fatalf("render html: %v", err)
	}
	if !strings.Contains(html, "<textarea name=\"symbols\">BTCUSDT") {
		t.Fatalf("expected textarea, got %q", html)
	}
	if !strings.Contains(html, "# QuantLab Dashboard") {
		t.Fatalf("expected markdown body, got %q", html)
	}
	if !strings.Contains(html, "## Market Context") || !strings.Contains(html, "fear_greed=11") {
		t.Fatalf("expected market context block, got %q", html)
	}
	if !strings.Contains(html, "保存 Watchlist") || !strings.Contains(html, "执行 Apply 并重启 marketd") {
		t.Fatalf("expected forms, got %q", html)
	}
}

type reporterStub struct {
	report insights.DashboardReport
	err    error
}

func (reporter reporterStub) BuildDashboard(context.Context) (insights.DashboardReport, error) {
	return reporter.report, reporter.err
}
