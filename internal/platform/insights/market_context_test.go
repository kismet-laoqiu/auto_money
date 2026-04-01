package insights

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
	"quantlab/internal/watchlist"
)

func TestServiceBuildMarketContextDerivesReturnsAndRelativeStrength(t *testing.T) {
	start := time.Unix(1710000000, 0).UTC()
	btcBars := buildBarsFromCloses(start, buildCloseSeries(100, 100))
	ethBars := buildBarsFromCloses(start, buildCloseSeries(50, 100))

	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 120},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 20, CooldownBars: 4},
		},
		Store: &storeStub{
			bars: map[string][]core.Bar{
				"bitget:BTCUSDT:1d": btcBars,
				"bitget:ETHUSDT:1d": ethBars,
			},
		},
		PriceReader: priceReaderMapStub{
			"BTCUSDT": btcBars[len(btcBars)-1].Close,
			"ETHUSDT": ethBars[len(ethBars)-1].Close,
		},
		Now: func() time.Time { return start.Add(99 * 24 * time.Hour) },
	})
	service.marketFetcher = marketContextFetcherStub{}

	snapshot := service.buildMarketContext(context.Background(), watchlist.File{
		Provider:    "bitget",
		ProductType: "USDT-FUTURES",
	})
	if snapshot == nil {
		t.Fatal("expected market context snapshot")
	}
	if snapshot.BTC == nil || snapshot.ETH == nil {
		t.Fatalf("expected btc and eth contexts, got %+v", snapshot)
	}
	if snapshot.RelativeStrength == nil {
		t.Fatalf("expected relative strength, got %+v", snapshot)
	}

	expectedBTC7d := percentChange(snapshot.BTC.CurrentPrice, btcBars[len(btcBars)-1-7].Close)
	expectedETH7d := percentChange(snapshot.ETH.CurrentPrice, ethBars[len(ethBars)-1-7].Close)
	expectedBTC30d := percentChange(snapshot.BTC.CurrentPrice, btcBars[len(btcBars)-1-30].Close)
	expectedETH30d := percentChange(snapshot.ETH.CurrentPrice, ethBars[len(ethBars)-1-30].Close)
	expectedBTC90d := percentChange(snapshot.BTC.CurrentPrice, btcBars[len(btcBars)-1-90].Close)
	expectedETH90d := percentChange(snapshot.ETH.CurrentPrice, ethBars[len(ethBars)-1-90].Close)

	assertClose(t, "btc 7d", snapshot.BTC.Return7d, expectedBTC7d)
	assertClose(t, "eth 7d", snapshot.ETH.Return7d, expectedETH7d)
	assertClose(t, "btc 30d", snapshot.BTC.Return30d, expectedBTC30d)
	assertClose(t, "eth 30d", snapshot.ETH.Return30d, expectedETH30d)
	assertClose(t, "btc 90d", snapshot.BTC.Return90d, expectedBTC90d)
	assertClose(t, "eth 90d", snapshot.ETH.Return90d, expectedETH90d)
	assertClose(t, "relative 7d", snapshot.RelativeStrength.BTCMinusETH7d, expectedBTC7d-expectedETH7d)
	assertClose(t, "relative 30d", snapshot.RelativeStrength.BTCMinusETH30d, expectedBTC30d-expectedETH30d)
	assertClose(t, "relative 90d", snapshot.RelativeStrength.BTCMinusETH90d, expectedBTC90d-expectedETH90d)
}

func TestMarketContextHTTPFetcherParsesExternalSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v3/ticker/24hr":
			_ = json.NewEncoder(writer).Encode(map[string]string{"lastPrice": "68000.00"})
		case "/api/v3/klines":
			payload := make([][]any, 0, 200)
			for i := 1; i <= 200; i++ {
				payload = append(payload, []any{
					float64(i),
					"1.0",
					"1.0",
					"1.0",
					float64ToString(float64(i)),
					"1.0",
				})
			}
			_ = json.NewEncoder(writer).Encode(payload)
		case "/api/v1/mining/hashrate/3d":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"currentHashrate": 1.011179630228448e+21,
			})
		case "/api/blocks/tip/height":
			_, _ = writer.Write([]byte("943022"))
		case "/fng/":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"data": []map[string]string{{
					"value":                "11",
					"value_classification": "Extreme Fear",
					"timestamp":            "1774915200",
				}},
			})
		case "/balancedPrice":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"code": 100,
				"data": []map[string]float64{
					{"t": 1774828800000, "v": 40321},
					{"t": 1774915200000, "v": 40458},
				},
			})
		case "/mCapRealizedRatio":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"code": 100,
				"data": []map[string]float64{
					{"t": 1774828800000, "v": 1.19},
					{"t": 1774915200000, "v": 1.23},
				},
			})
		case "/mnav":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"mstr": map[string]float64{
					"btc_holdings": 762099,
					"debt":         8253923000,
					"pref":         10009094200,
					"cash":         2250000000,
					"shares":       345594000,
					"stock_price":  121.44,
				},
				"bmnr": map[string]float64{
					"shares":       478850823,
					"cash":         586000000,
					"eth_holdings": 4285126,
					"stock_price":  18.3,
				},
				"eth_price": 2057.64,
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	fetcher := marketContextHTTPFetcher{
		client:           server.Client(),
		now:              func() time.Time { return time.Unix(1774915200, 0).UTC() },
		tickerURL:        server.URL + "/api/v3/ticker/24hr",
		weeklyKlinesURL:  server.URL + "/api/v3/klines",
		hashrateURL:      server.URL + "/api/v1/mining/hashrate/3d",
		tipHeightURL:     server.URL + "/api/blocks/tip/height",
		fearGreedURL:     server.URL + "/fng/",
		balancedPriceURL: server.URL + "/balancedPrice",
		mvrvURL:          server.URL + "/mCapRealizedRatio",
		mnavURL:          server.URL + "/mnav",
	}

	snapshot, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch external market context: %v", err)
	}

	assertClose(t, "btc price", snapshot.BTCPrice, 68000)
	assertClose(t, "200wma", snapshot.WMA200, 100.5)
	if snapshot.FearGreed == nil || snapshot.FearGreed.Value != 11 || snapshot.FearGreed.Classification != "Extreme Fear" {
		t.Fatalf("unexpected fear and greed: %+v", snapshot.FearGreed)
	}
	if snapshot.Hashrate == nil {
		t.Fatal("expected hashrate snapshot")
	}
	assertClose(t, "hashrate eh", snapshot.Hashrate.CurrentEH, 1011.179630228448)
	if snapshot.Halving == nil || snapshot.Halving.TargetBlock != 1050000 || snapshot.Halving.BlocksRemaining != 106978 {
		t.Fatalf("unexpected halving snapshot: %+v", snapshot.Halving)
	}
	assertClose(t, "current reward", snapshot.Halving.CurrentReward, 3.125)
	assertClose(t, "next reward", snapshot.Halving.NextReward, 1.5625)
	if snapshot.BalancedPrice == nil || snapshot.BalancedPrice.Value != 40458 {
		t.Fatalf("unexpected balanced price: %+v", snapshot.BalancedPrice)
	}
	if snapshot.MVRV == nil || snapshot.MVRV.Value != 1.23 {
		t.Fatalf("unexpected mvrv: %+v", snapshot.MVRV)
	}
	if snapshot.Mnav == nil || snapshot.Mnav.MSTR == nil || snapshot.Mnav.BMNR == nil {
		t.Fatalf("unexpected mnav snapshot: %+v", snapshot.Mnav)
	}
	assertClose(t, "mstr basic", snapshot.Mnav.MSTR.BasicRatio, (345594000*121.44)/(762099*68000))
	assertClose(t, "mstr enterprise", snapshot.Mnav.MSTR.EnterpriseRatio, ((345594000*121.44)+8253923000+10009094200-2250000000)/(762099*68000))
	assertClose(t, "bmnr ratio", snapshot.Mnav.BMNR.Ratio, (478850823*18.3)/(4285126*2057.64))
}

func TestServiceBuildMarketContextDegradesWhenExternalFetchFails(t *testing.T) {
	start := time.Unix(1710000000, 0).UTC()
	btcBars := buildBarsFromCloses(start, buildCloseSeries(100, 100))
	ethBars := buildBarsFromCloses(start, buildCloseSeries(50, 100))

	service := NewService(Config{
		WatchlistPath: "configs/platform/watchlist.yaml",
		Strategy:      config.StrategyConfig{FastSMA: 2, SlowSMA: 3, ATRWindow: 2, LevelLookback: 2, PivotWindow: 1},
		Insights: config.InsightsConfig{
			DailySignal: config.DailySignalAlertConfig{Interval: "1d", LookbackBars: 120},
			Anomaly:     config.AnomalyAlertConfig{Interval: "15m", RollingWindow: 20, CooldownBars: 4},
		},
		Store: &storeStub{
			bars: map[string][]core.Bar{
				"bitget:BTCUSDT:1d": btcBars,
				"bitget:ETHUSDT:1d": ethBars,
			},
		},
		PriceReader: priceReaderMapStub{
			"BTCUSDT": btcBars[len(btcBars)-1].Close,
			"ETHUSDT": ethBars[len(ethBars)-1].Close,
		},
		Now: func() time.Time { return start.Add(99 * 24 * time.Hour) },
	})
	service.marketFetcher = marketContextFetcherStub{err: errors.New("upstream unavailable")}

	snapshot := service.buildMarketContext(context.Background(), watchlist.File{
		Provider:    "bitget",
		ProductType: "USDT-FUTURES",
	})
	if snapshot == nil || snapshot.BTC == nil || snapshot.ETH == nil {
		t.Fatalf("expected warehouse-derived snapshot, got %+v", snapshot)
	}
	if snapshot.FearGreed != nil || snapshot.Hashrate != nil || snapshot.BalancedPrice != nil || snapshot.MVRV != nil || snapshot.Mnav != nil {
		t.Fatalf("expected external fields to degrade to nil, got %+v", snapshot)
	}
}

type priceReaderMapStub map[string]float64

func (stub priceReaderMapStub) FetchTickerPrice(_ context.Context, symbol, _ string) (float64, error) {
	return stub[symbol], nil
}

func buildCloseSeries(start float64, count int) []float64 {
	out := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, start+float64(i))
	}
	return out
}

func buildBarsFromCloses(start time.Time, closes []float64) []core.Bar {
	out := make([]core.Bar, 0, len(closes))
	for i, closePrice := range closes {
		out = append(out, core.Bar{
			Time:   start.Add(time.Duration(i) * 24 * time.Hour),
			Open:   closePrice - 1,
			High:   closePrice + 1,
			Low:    closePrice - 2,
			Close:  closePrice,
			Volume: 10,
		})
	}
	return out
}

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("%s mismatch: got %.10f want %.10f", label, got, want)
	}
}

func float64ToString(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
