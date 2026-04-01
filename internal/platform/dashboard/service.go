package dashboard

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"

	"quantlab/internal/platform/insights"
)

type InsightsReporter interface {
	BuildDashboard(ctx context.Context) (insights.DashboardReport, error)
}

type Config struct {
	Insights InsightsReporter
}

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

func (service *Service) Report(ctx context.Context) (insights.DashboardReport, error) {
	if service == nil || service.cfg.Insights == nil {
		return insights.DashboardReport{}, fmt.Errorf("dashboard insights reporter is nil")
	}
	return service.cfg.Insights.BuildDashboard(ctx)
}

func (service *Service) RenderHTML(ctx context.Context, symbolsText string, flash string) (string, error) {
	report, err := service.Report(ctx)
	if err != nil {
		return "", err
	}
	return renderHTML(report, symbolsText, flash)
}

func renderHTML(report insights.DashboardReport, symbolsText string, flash string) (string, error) {
	view := struct {
		Flash     string
		Markdown  string
		Symbols   string
		Generated string
	}{
		Flash:     flash,
		Markdown:  buildMarkdown(report),
		Symbols:   symbolsText,
		Generated: report.GeneratedAt.Format("2006-01-02 15:04:05 MST"),
	}
	var builder strings.Builder
	if err := pageTemplate.Execute(&builder, view); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func buildMarkdown(report insights.DashboardReport) string {
	var builder strings.Builder
	builder.WriteString("# QuantLab Dashboard\n\n")
	builder.WriteString("## Watchlist\n")
	for _, symbol := range report.Symbols {
		builder.WriteString(fmt.Sprintf("- `%s` latest=%.4f rsi14=%.2f volume_zscore=%.2f trend_up=%t\n",
			symbol.Symbol,
			symbol.LatestPrice,
			symbol.DailyFeatures.RSI14,
			symbol.DailyFeatures.VolumeZScore,
			symbol.DailyFeatures.TrendUp,
		))
		if symbol.DailySignal != nil {
			builder.WriteString(fmt.Sprintf("  daily_signal=%s score=%.2f threshold=%.2f\n",
				symbol.DailySignal.Side,
				symbol.DailySignal.Score,
				symbol.DailySignal.Threshold,
			))
		}
		if symbol.LatestMarketAlert != nil {
			builder.WriteString(fmt.Sprintf("  market_alert=%s move_pct=%.2f volume_ratio=%.2f\n",
				symbol.LatestMarketAlert.Signal,
				symbol.LatestMarketAlert.MovePct,
				symbol.LatestMarketAlert.VolumeRatio,
			))
		}
	}
	writeMarketContextMarkdown(&builder, report.MarketContext)
	builder.WriteString("\n## Active Alerts\n")
	if len(report.Alerts) == 0 {
		builder.WriteString("- none\n")
		return builder.String()
	}
	for _, alert := range report.Alerts {
		builder.WriteString(fmt.Sprintf("- [%s] %s | %s\n", alert.Interval, alert.Title, alert.Summary))
	}
	return builder.String()
}

func writeMarketContextMarkdown(builder *strings.Builder, snapshot *insights.MarketContextSnapshot) {
	if snapshot == nil {
		return
	}
	builder.WriteString("\n## Market Context\n")
	wrote := false
	if snapshot.BTC != nil {
		builder.WriteString(fmt.Sprintf("- btc price=%.2f wma200=%.2f price_to_wma200=%.2fx return_7d=%.2f return_30d=%.2f return_90d=%.2f\n",
			snapshot.BTC.CurrentPrice,
			snapshot.BTC.WMA200,
			snapshot.BTC.PriceToWMA200,
			snapshot.BTC.Return7d,
			snapshot.BTC.Return30d,
			snapshot.BTC.Return90d,
		))
		wrote = true
	}
	if snapshot.ETH != nil {
		builder.WriteString(fmt.Sprintf("- eth price=%.2f return_7d=%.2f return_30d=%.2f return_90d=%.2f\n",
			snapshot.ETH.CurrentPrice,
			snapshot.ETH.Return7d,
			snapshot.ETH.Return30d,
			snapshot.ETH.Return90d,
		))
		wrote = true
	}
	if snapshot.RelativeStrength != nil {
		builder.WriteString(fmt.Sprintf("- btc_minus_eth_7d=%.2f btc_minus_eth_30d=%.2f btc_minus_eth_90d=%.2f\n",
			snapshot.RelativeStrength.BTCMinusETH7d,
			snapshot.RelativeStrength.BTCMinusETH30d,
			snapshot.RelativeStrength.BTCMinusETH90d,
		))
		wrote = true
	}
	if snapshot.FearGreed != nil {
		builder.WriteString(fmt.Sprintf("- fear_greed=%d (%s)\n", snapshot.FearGreed.Value, snapshot.FearGreed.Classification))
		wrote = true
	}
	if snapshot.Hashrate != nil {
		builder.WriteString(fmt.Sprintf("- hashrate_eh=%.2f\n", snapshot.Hashrate.CurrentEH))
		wrote = true
	}
	if snapshot.Halving != nil {
		builder.WriteString(fmt.Sprintf("- halving_block=%d target_block=%d days_remaining=%.2f\n",
			snapshot.Halving.CurrentBlock,
			snapshot.Halving.TargetBlock,
			snapshot.Halving.DaysRemaining,
		))
		wrote = true
	}
	if snapshot.BalancedPrice != nil {
		builder.WriteString(fmt.Sprintf("- balanced_price=%.2f\n", snapshot.BalancedPrice.Value))
		wrote = true
	}
	if snapshot.MVRV != nil {
		builder.WriteString(fmt.Sprintf("- mvrv=%.2f\n", snapshot.MVRV.Value))
		wrote = true
	}
	if snapshot.Mnav != nil {
		if snapshot.Mnav.MSTR != nil {
			builder.WriteString(fmt.Sprintf("- mstr basic_ratio=%.2fx enterprise_ratio=%.2fx stock_price=%.2f holdings=%.0f\n",
				snapshot.Mnav.MSTR.BasicRatio,
				snapshot.Mnav.MSTR.EnterpriseRatio,
				snapshot.Mnav.MSTR.StockPrice,
				snapshot.Mnav.MSTR.Holdings,
			))
			wrote = true
		}
		if snapshot.Mnav.BMNR != nil {
			builder.WriteString(fmt.Sprintf("- bmnr ratio=%.2fx stock_price=%.2f holdings=%.0f eth_price=%.2f\n",
				snapshot.Mnav.BMNR.Ratio,
				snapshot.Mnav.BMNR.StockPrice,
				snapshot.Mnav.BMNR.Holdings,
				snapshot.Mnav.ETHPrice,
			))
			wrote = true
		}
	}
	if !wrote {
		builder.WriteString("- none\n")
	}
}

var pageTemplate = template.Must(template.New("dashboard").Parse(`
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>QuantLab Dashboard</title>
  <style>
    body { font-family: Menlo, Monaco, Consolas, monospace; margin: 24px; line-height: 1.5; background: #f7f4ee; color: #1d1d1d; }
    h1, h2 { margin: 0 0 12px; }
    .flash { padding: 10px 12px; background: #fff4c2; border: 1px solid #d0b44a; margin-bottom: 16px; }
    .actions { display: grid; gap: 16px; margin: 16px 0 24px; }
    textarea { width: 100%; min-height: 180px; font: inherit; }
    button { font: inherit; padding: 8px 14px; }
    pre { white-space: pre-wrap; word-break: break-word; background: #fff; border: 1px solid #ddd2c7; padding: 16px; }
  </style>
</head>
<body>
  <h1>QuantLab Dashboard</h1>
  <p>Generated: {{ .Generated }}</p>
  {{ if .Flash }}<div class="flash">{{ .Flash }}</div>{{ end }}
  <div class="actions">
    <form method="post" action="/api/watchlist/save">
      <h2>Watchlist Symbols</h2>
      <textarea name="symbols">{{ .Symbols }}</textarea>
      <div><button type="submit">保存 Watchlist</button></div>
    </form>
    <form method="post" action="/api/watchlist/apply">
      <h2>Apply</h2>
      <div><button type="submit">执行 Apply 并重启 marketd</button></div>
    </form>
  </div>
  <pre>{{ .Markdown }}</pre>
</body>
</html>
`))

func WriteHTML(writer io.Writer, html string) error {
	_, err := io.WriteString(writer, html)
	return err
}
