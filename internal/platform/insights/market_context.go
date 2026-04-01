package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"quantlab/internal/core"
	"quantlab/internal/watchlist"
)

const (
	defaultBTCTickerURL         = "https://api.binance.com/api/v3/ticker/24hr?symbol=BTCUSDT"
	defaultBTCWeeklyKlinesURL   = "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1w&limit=200"
	defaultHashrateURL          = "https://mempool.space/api/v1/mining/hashrate/3d"
	defaultTipHeightURL         = "https://mempool.space/api/blocks/tip/height"
	defaultFearGreedURL         = "https://api.alternative.me/fng/?limit=1"
	defaultBalancedPriceURL     = "https://looknode-proxy.corms-cushier-0l.workers.dev/balancedPrice"
	defaultMVRVURL              = "https://looknode-proxy.corms-cushier-0l.workers.dev/mCapRealizedRatio"
	defaultMNAVURL              = "https://looknode-proxy.corms-cushier-0l.workers.dev/mnav"
	marketContextLookupLookback = 91
)

type marketContextFetcher interface {
	Fetch(ctx context.Context) (marketContextExternalSnapshot, error)
}

type marketContextExternalSnapshot struct {
	BTCPrice      float64
	WMA200        float64
	FearGreed     *FearGreedSnapshot
	Hashrate      *HashrateSnapshot
	Halving       *HalvingSnapshot
	BalancedPrice *MarketLevelSnapshot
	MVRV          *MarketLevelSnapshot
	Mnav          *TreasuryPremiumSnapshot
}

type marketContextHTTPFetcher struct {
	client           *http.Client
	now              func() time.Time
	tickerURL        string
	weeklyKlinesURL  string
	hashrateURL      string
	tipHeightURL     string
	fearGreedURL     string
	balancedPriceURL string
	mvrvURL          string
	mnavURL          string
}

func newMarketContextHTTPFetcher(now func() time.Time) marketContextFetcher {
	if now == nil {
		now = time.Now
	}
	return marketContextHTTPFetcher{
		client: &http.Client{Timeout: 15 * time.Second},
		now:    now,

		tickerURL:        defaultBTCTickerURL,
		weeklyKlinesURL:  defaultBTCWeeklyKlinesURL,
		hashrateURL:      defaultHashrateURL,
		tipHeightURL:     defaultTipHeightURL,
		fearGreedURL:     defaultFearGreedURL,
		balancedPriceURL: defaultBalancedPriceURL,
		mvrvURL:          defaultMVRVURL,
		mnavURL:          defaultMNAVURL,
	}
}

func (service *Service) buildMarketContext(ctx context.Context, file watchlist.File) *MarketContextSnapshot {
	if cached := service.loadCachedMarketContext(); cached != nil {
		return cached
	}

	snapshot := &MarketContextSnapshot{
		BTC: service.loadMarketAsset(ctx, file, "BTCUSDT"),
		ETH: service.loadMarketAsset(ctx, file, "ETHUSDT"),
	}
	if snapshot.BTC != nil && snapshot.ETH != nil {
		snapshot.RelativeStrength = &RelativeStrengthSnapshot{
			BTCMinusETH7d:  snapshot.BTC.Return7d - snapshot.ETH.Return7d,
			BTCMinusETH30d: snapshot.BTC.Return30d - snapshot.ETH.Return30d,
			BTCMinusETH90d: snapshot.BTC.Return90d - snapshot.ETH.Return90d,
		}
	}

	if service.marketFetcher != nil {
		external, _ := service.marketFetcher.Fetch(ctx)
		mergeMarketContextExternal(snapshot, external)
	}

	if isEmptyMarketContext(snapshot) {
		return nil
	}
	service.storeCachedMarketContext(snapshot)
	return snapshot
}

func (service *Service) loadCachedMarketContext() *MarketContextSnapshot {
	if service == nil || service.cfg.Insights.PriceTTL <= 0 {
		return nil
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.marketCache.Snapshot == nil {
		return nil
	}
	if service.cfg.Now().Sub(service.marketCache.At) > service.cfg.Insights.PriceTTL {
		return nil
	}
	return cloneMarketContextSnapshot(service.marketCache.Snapshot)
}

func (service *Service) storeCachedMarketContext(snapshot *MarketContextSnapshot) {
	if service == nil || service.cfg.Insights.PriceTTL <= 0 || snapshot == nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.marketCache = cachedMarketContext{
		At:       service.cfg.Now(),
		Snapshot: cloneMarketContextSnapshot(snapshot),
	}
}

func (service *Service) loadMarketAsset(ctx context.Context, file watchlist.File, symbol string) *MarketAssetSnapshot {
	if service == nil || service.cfg.Store == nil {
		return nil
	}
	limit := maxInt(service.cfg.Insights.DailySignal.LookbackBars, marketContextLookupLookback)
	bars, err := service.cfg.Store.LoadRecentBars(ctx, file.Provider, symbol, service.cfg.Insights.DailySignal.Interval, limit)
	if err != nil || len(bars) == 0 {
		return nil
	}
	latest := bars[len(bars)-1]
	currentPrice := latest.Close
	if service.cfg.PriceReader != nil {
		if price, priceErr := service.cfg.PriceReader.FetchTickerPrice(ctx, symbol, file.ProductType); priceErr == nil && price > 0 {
			currentPrice = price
		}
	}
	return &MarketAssetSnapshot{
		Symbol:       symbol,
		CurrentPrice: currentPrice,
		Return7d:     percentChange(currentPrice, closeAtLookback(bars, 7)),
		Return30d:    percentChange(currentPrice, closeAtLookback(bars, 30)),
		Return90d:    percentChange(currentPrice, closeAtLookback(bars, 90)),
		UpdatedAt:    latest.Time.UTC(),
	}
}

func mergeMarketContextExternal(snapshot *MarketContextSnapshot, external marketContextExternalSnapshot) {
	if snapshot == nil {
		return
	}
	if external.BTCPrice > 0 {
		if snapshot.BTC == nil {
			snapshot.BTC = &MarketAssetSnapshot{Symbol: "BTCUSDT"}
		}
		if snapshot.BTC.CurrentPrice == 0 {
			snapshot.BTC.CurrentPrice = external.BTCPrice
		}
	}
	if external.WMA200 > 0 {
		if snapshot.BTC == nil {
			snapshot.BTC = &MarketAssetSnapshot{Symbol: "BTCUSDT"}
		}
		snapshot.BTC.WMA200 = external.WMA200
		if snapshot.BTC.CurrentPrice > 0 {
			snapshot.BTC.PriceToWMA200 = snapshot.BTC.CurrentPrice / external.WMA200
		}
	}
	if external.FearGreed != nil {
		snapshot.FearGreed = external.FearGreed
	}
	if external.Hashrate != nil {
		snapshot.Hashrate = external.Hashrate
	}
	if external.Halving != nil {
		snapshot.Halving = external.Halving
	}
	if external.BalancedPrice != nil {
		snapshot.BalancedPrice = external.BalancedPrice
	}
	if external.MVRV != nil {
		snapshot.MVRV = external.MVRV
	}
	if external.Mnav != nil {
		snapshot.Mnav = external.Mnav
	}
}

func isEmptyMarketContext(snapshot *MarketContextSnapshot) bool {
	return snapshot == nil ||
		(snapshot.BTC == nil &&
			snapshot.ETH == nil &&
			snapshot.RelativeStrength == nil &&
			snapshot.FearGreed == nil &&
			snapshot.Hashrate == nil &&
			snapshot.Halving == nil &&
			snapshot.BalancedPrice == nil &&
			snapshot.MVRV == nil &&
			snapshot.Mnav == nil)
}

func cloneMarketContextSnapshot(snapshot *MarketContextSnapshot) *MarketContextSnapshot {
	if snapshot == nil {
		return nil
	}
	cloned := *snapshot
	return &cloned
}

func percentChange(current, base float64) float64 {
	if current <= 0 || base <= 0 {
		return 0
	}
	return (current/base - 1) * 100
}

func closeAtLookback(bars []core.Bar, lookback int) float64 {
	if len(bars) <= lookback {
		return 0
	}
	return bars[len(bars)-1-lookback].Close
}

func (fetcher marketContextHTTPFetcher) Fetch(ctx context.Context) (marketContextExternalSnapshot, error) {
	var out marketContextExternalSnapshot
	var firstErr error
	setErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	price, err := fetcher.fetchTickerPrice(ctx)
	setErr(err)
	out.BTCPrice = price

	wma, err := fetcher.fetchWMA200(ctx)
	setErr(err)
	out.WMA200 = wma

	out.FearGreed, err = fetcher.fetchFearGreed(ctx)
	setErr(err)

	out.Hashrate, err = fetcher.fetchHashrate(ctx)
	setErr(err)

	out.Halving, err = fetcher.fetchHalving(ctx)
	setErr(err)

	out.BalancedPrice, err = fetcher.fetchSeriesMetric(ctx, fetcher.balancedPriceURL)
	setErr(err)

	out.MVRV, err = fetcher.fetchSeriesMetric(ctx, fetcher.mvrvURL)
	setErr(err)

	out.Mnav, err = fetcher.fetchMnav(ctx, out.BTCPrice)
	setErr(err)

	return out, firstErr
}

func (fetcher marketContextHTTPFetcher) fetchTickerPrice(ctx context.Context) (float64, error) {
	var payload struct {
		LastPrice string `json:"lastPrice"`
	}
	if err := fetcher.fetchJSON(ctx, fetcher.tickerURL, &payload); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(payload.LastPrice, 64)
}

func (fetcher marketContextHTTPFetcher) fetchWMA200(ctx context.Context) (float64, error) {
	var payload [][]any
	if err := fetcher.fetchJSON(ctx, fetcher.weeklyKlinesURL, &payload); err != nil {
		return 0, err
	}
	if len(payload) < 200 {
		return 0, fmt.Errorf("weekly klines insufficient: %d", len(payload))
	}
	sum := 0.0
	for _, item := range payload[len(payload)-200:] {
		if len(item) < 5 {
			return 0, fmt.Errorf("weekly klines payload malformed")
		}
		closePrice, err := parseFloatValue(item[4])
		if err != nil {
			return 0, err
		}
		sum += closePrice
	}
	return sum / 200, nil
}

func (fetcher marketContextHTTPFetcher) fetchFearGreed(ctx context.Context) (*FearGreedSnapshot, error) {
	var payload struct {
		Data []struct {
			Value               string `json:"value"`
			ValueClassification string `json:"value_classification"`
			Timestamp           string `json:"timestamp"`
		} `json:"data"`
	}
	if err := fetcher.fetchJSON(ctx, fetcher.fearGreedURL, &payload); err != nil {
		return nil, err
	}
	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("fear and greed payload empty")
	}
	value, err := strconv.Atoi(payload.Data[0].Value)
	if err != nil {
		return nil, err
	}
	ts, err := strconv.ParseInt(payload.Data[0].Timestamp, 10, 64)
	if err != nil {
		return nil, err
	}
	return &FearGreedSnapshot{
		Value:          value,
		Classification: payload.Data[0].ValueClassification,
		UpdatedAt:      time.Unix(ts, 0).UTC(),
	}, nil
}

func (fetcher marketContextHTTPFetcher) fetchHashrate(ctx context.Context) (*HashrateSnapshot, error) {
	var payload struct {
		CurrentHashrate float64 `json:"currentHashrate"`
		Hashrates       []struct {
			Timestamp int64 `json:"timestamp"`
		} `json:"hashrates"`
	}
	if err := fetcher.fetchJSON(ctx, fetcher.hashrateURL, &payload); err != nil {
		return nil, err
	}
	if payload.CurrentHashrate <= 0 {
		return nil, fmt.Errorf("hashrate unavailable")
	}
	updatedAt := fetcher.now().UTC()
	if n := len(payload.Hashrates); n > 0 && payload.Hashrates[n-1].Timestamp > 0 {
		updatedAt = time.Unix(payload.Hashrates[n-1].Timestamp, 0).UTC()
	}
	return &HashrateSnapshot{
		CurrentEH: payload.CurrentHashrate / 1e18,
		UpdatedAt: updatedAt,
	}, nil
}

func (fetcher marketContextHTTPFetcher) fetchHalving(ctx context.Context) (*HalvingSnapshot, error) {
	body, err := fetcher.fetchText(ctx, fetcher.tipHeightURL)
	if err != nil {
		return nil, err
	}
	height, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return nil, err
	}
	halvingsCompleted := height / 210000
	target := (halvingsCompleted + 1) * 210000
	remaining := target - height
	if remaining < 0 {
		remaining = 0
	}
	currentReward := 50.0 / float64(int64(1)<<uint(halvingsCompleted))
	return &HalvingSnapshot{
		CurrentBlock:    height,
		TargetBlock:     target,
		BlocksRemaining: remaining,
		DaysRemaining:   float64(remaining) * 10 / 60 / 24,
		CurrentReward:   currentReward,
		NextReward:      currentReward / 2,
		EstimatedAt:     fetcher.now().UTC().Add(time.Duration(remaining) * 10 * time.Minute),
	}, nil
}

func (fetcher marketContextHTTPFetcher) fetchSeriesMetric(ctx context.Context, targetURL string) (*MarketLevelSnapshot, error) {
	var payload struct {
		Code int `json:"code"`
		Data []struct {
			T float64 `json:"t"`
			V float64 `json:"v"`
		} `json:"data"`
	}
	if err := fetcher.fetchJSON(ctx, targetURL, &payload); err != nil {
		return nil, err
	}
	for i := len(payload.Data) - 1; i >= 0; i-- {
		if payload.Data[i].V <= 0 {
			continue
		}
		return &MarketLevelSnapshot{
			Value:     payload.Data[i].V,
			UpdatedAt: time.UnixMilli(int64(payload.Data[i].T)).UTC(),
		}, nil
	}
	return nil, fmt.Errorf("series payload empty")
}

func (fetcher marketContextHTTPFetcher) fetchMnav(ctx context.Context, btcPrice float64) (*TreasuryPremiumSnapshot, error) {
	var payload struct {
		MSTR struct {
			BTCHoldings float64 `json:"btc_holdings"`
			Debt        float64 `json:"debt"`
			Pref        float64 `json:"pref"`
			Cash        float64 `json:"cash"`
			Shares      float64 `json:"shares"`
			StockPrice  float64 `json:"stock_price"`
		} `json:"mstr"`
		BMNR struct {
			Shares      float64 `json:"shares"`
			Cash        float64 `json:"cash"`
			ETHHoldings float64 `json:"eth_holdings"`
			StockPrice  float64 `json:"stock_price"`
		} `json:"bmnr"`
		ETHPrice float64 `json:"eth_price"`
	}
	if err := fetcher.fetchJSON(ctx, fetcher.mnavURL, &payload); err != nil {
		return nil, err
	}
	snapshot := &TreasuryPremiumSnapshot{
		ETHPrice: payload.ETHPrice,
	}
	if payload.MSTR.BTCHoldings > 0 && payload.MSTR.Shares > 0 && payload.MSTR.StockPrice > 0 {
		marketCap := payload.MSTR.Shares * payload.MSTR.StockPrice
		company := &TreasuryCompanySnapshot{
			Symbol:     "MSTR",
			Holdings:   payload.MSTR.BTCHoldings,
			StockPrice: payload.MSTR.StockPrice,
		}
		if btcPrice > 0 {
			assetValue := payload.MSTR.BTCHoldings * btcPrice
			company.BasicRatio = marketCap / assetValue
			company.EnterpriseRatio = (marketCap + payload.MSTR.Debt + payload.MSTR.Pref - payload.MSTR.Cash) / assetValue
		}
		snapshot.MSTR = company
	}
	if payload.BMNR.ETHHoldings > 0 && payload.BMNR.Shares > 0 && payload.BMNR.StockPrice > 0 {
		marketCap := payload.BMNR.Shares * payload.BMNR.StockPrice
		company := &TreasuryCompanySnapshot{
			Symbol:     "BMNR",
			Holdings:   payload.BMNR.ETHHoldings,
			StockPrice: payload.BMNR.StockPrice,
		}
		if payload.ETHPrice > 0 {
			company.Ratio = marketCap / (payload.BMNR.ETHHoldings * payload.ETHPrice)
		}
		snapshot.BMNR = company
	}
	if snapshot.MSTR == nil && snapshot.BMNR == nil && snapshot.ETHPrice == 0 {
		return nil, fmt.Errorf("mnav payload empty")
	}
	return snapshot, nil
}

func (fetcher marketContextHTTPFetcher) fetchJSON(ctx context.Context, targetURL string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	response, err := fetcher.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("market context status=%d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (fetcher marketContextHTTPFetcher) fetchText(ctx context.Context, targetURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := fetcher.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("market context status=%d", response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

func parseFloatValue(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case string:
		return strconv.ParseFloat(typed, 64)
	case json.Number:
		return typed.Float64()
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}
