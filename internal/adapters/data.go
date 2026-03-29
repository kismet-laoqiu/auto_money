package adapters

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (client *Client) EnsureDataset(ctx context.Context, cacheDir string, spec config.DatasetConfig, refresh bool) (core.Dataset, error) {
	cachePath := datasetCachePath(cacheDir, spec)
	if !refresh {
		if bars, err := readBarsCSV(cachePath); err == nil && len(bars) > 0 {
			bars = trimBarsByRange(bars, spec.Range)
			return core.Dataset{Name: spec.Name, Provider: spec.Provider, Symbol: spec.Symbol, Interval: spec.Interval, Bars: bars}, nil
		}
	}
	bars, err := client.FetchBars(ctx, spec)
	if err != nil {
		return core.Dataset{}, err
	}
	if err := writeBarsCSV(cachePath, bars); err != nil {
		return core.Dataset{}, err
	}
	bars = trimBarsByRange(bars, spec.Range)
	return core.Dataset{Name: spec.Name, Provider: spec.Provider, Symbol: spec.Symbol, Interval: spec.Interval, Bars: bars}, nil
}

func (client *Client) FetchBars(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	switch strings.ToLower(spec.Provider) {
	case "binance":
		return client.fetchBinance(ctx, spec)
	case "bitget":
		return client.fetchBitget(ctx, spec)
	case "yahoo":
		return client.fetchYahoo(ctx, spec)
	case "stooq":
		return client.fetchStooq(ctx, spec)
	default:
		return nil, fmt.Errorf("unsupported provider %s", spec.Provider)
	}
}

func (client *Client) fetchBinance(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	limit := spec.Limit
	if limit <= 0 {
		limit = 1000
	}
	endpoint := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%d", url.QueryEscape(strings.ToUpper(spec.Symbol)), url.QueryEscape(strings.ToLower(spec.Interval)), limit)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	body, err := client.doRequest(request)
	if err != nil {
		return nil, err
	}
	var payload [][]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode binance %s: %w", spec.Symbol, err)
	}
	bars := make([]core.Bar, 0, len(payload))
	for _, row := range payload {
		if len(row) < 6 {
			continue
		}
		openTime, err := asInt64(row[0])
		if err != nil {
			return nil, err
		}
		openValue, _ := asFloat64(row[1])
		highValue, _ := asFloat64(row[2])
		lowValue, _ := asFloat64(row[3])
		closeValue, _ := asFloat64(row[4])
		volumeValue, _ := asFloat64(row[5])
		bars = append(bars, core.Bar{Time: time.UnixMilli(openTime).UTC(), Open: openValue, High: highValue, Low: lowValue, Close: closeValue, Volume: volumeValue})
	}
	return bars, nil
}

func (client *Client) fetchBitget(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	limit := spec.Limit
	if limit <= 0 {
		limit = 1000
	}
	endpoint := fmt.Sprintf("https://api.bitget.com/api/v2/spot/market/candles?symbol=%s&granularity=%s&limit=%d", url.QueryEscape(strings.ToUpper(spec.Symbol)), url.QueryEscape(bitgetGranularity(spec.Interval)), limit)
	if spec.ProductType != "" {
		endpoint = fmt.Sprintf(
			"https://api.bitget.com/api/v2/mix/market/candles?symbol=%s&productType=%s&granularity=%s&limit=%d",
			url.QueryEscape(strings.ToUpper(spec.Symbol)),
			url.QueryEscape(spec.ProductType),
			url.QueryEscape(bitgetGranularity(spec.Interval)),
			limit,
		)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	body, err := client.doRequest(request)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data [][]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode bitget %s: %w", spec.Symbol, err)
	}
	bars := make([]core.Bar, 0, len(payload.Data))
	for _, row := range payload.Data {
		if len(row) < 6 {
			continue
		}
		ts, err := asInt64(row[0])
		if err != nil {
			return nil, err
		}
		openValue, _ := asFloat64(row[1])
		highValue, _ := asFloat64(row[2])
		lowValue, _ := asFloat64(row[3])
		closeValue, _ := asFloat64(row[4])
		volumeValue, _ := asFloat64(row[5])
		bars = append(bars, core.Bar{Time: time.UnixMilli(ts).UTC(), Open: openValue, High: highValue, Low: lowValue, Close: closeValue, Volume: volumeValue})
	}
	sort.Slice(bars, func(i, j int) bool { return bars[i].Time.Before(bars[j].Time) })
	return bars, nil
}

func (client *Client) fetchYahoo(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	rangeValue := spec.Range
	if rangeValue == "" {
		rangeValue = "2y"
	}
	intervalValue := spec.Interval
	if intervalValue == "" {
		intervalValue = "1d"
	}
	endpoint := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s&includePrePost=false&events=div%%2Csplits", url.PathEscape(spec.Symbol), url.QueryEscape(intervalValue), url.QueryEscape(rangeValue))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "quant-lab/0.1")
	body, err := client.doRequest(request)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Open   []*float64 `json:"open"`
						High   []*float64 `json:"high"`
						Low    []*float64 `json:"low"`
						Close  []*float64 `json:"close"`
						Volume []*float64 `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode yahoo %s: %w", spec.Symbol, err)
	}
	if len(payload.Chart.Result) == 0 || len(payload.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo returned no bars for %s", spec.Symbol)
	}
	result := payload.Chart.Result[0]
	quote := result.Indicators.Quote[0]
	bars := make([]core.Bar, 0, len(result.Timestamp))
	for index, ts := range result.Timestamp {
		if index >= len(quote.Open) || index >= len(quote.High) || index >= len(quote.Low) || index >= len(quote.Close) {
			continue
		}
		if quote.Open[index] == nil || quote.High[index] == nil || quote.Low[index] == nil || quote.Close[index] == nil {
			continue
		}
		volume := 0.0
		if index < len(quote.Volume) && quote.Volume[index] != nil {
			volume = *quote.Volume[index]
		}
		bars = append(bars, core.Bar{Time: time.Unix(ts, 0).UTC(), Open: *quote.Open[index], High: *quote.High[index], Low: *quote.Low[index], Close: *quote.Close[index], Volume: volume})
	}
	return bars, nil
}

func (client *Client) fetchStooq(ctx context.Context, spec config.DatasetConfig) ([]core.Bar, error) {
	endpoint := fmt.Sprintf("https://stooq.com/q/d/l/?s=%s&i=%s", url.QueryEscape(strings.ToLower(spec.Symbol)), stooqInterval(spec.Interval))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	body, err := client.doRequest(request)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	bars := make([]core.Bar, 0, len(records)-1)
	for index, record := range records {
		if index == 0 || len(record) < 5 {
			continue
		}
		timestamp, err := time.Parse("2006-01-02", record[0])
		if err != nil {
			continue
		}
		openValue, _ := strconv.ParseFloat(record[1], 64)
		highValue, _ := strconv.ParseFloat(record[2], 64)
		lowValue, _ := strconv.ParseFloat(record[3], 64)
		closeValue, _ := strconv.ParseFloat(record[4], 64)
		volumeValue := 0.0
		if len(record) > 5 {
			volumeValue, _ = strconv.ParseFloat(record[5], 64)
		}
		bars = append(bars, core.Bar{Time: timestamp.UTC(), Open: openValue, High: highValue, Low: lowValue, Close: closeValue, Volume: volumeValue})
	}
	bars = trimBarsByRange(bars, spec.Range)
	return bars, nil
}

func trimBarsByRange(bars []core.Bar, rangeValue string) []core.Bar {
	if len(bars) == 0 || rangeValue == "" {
		return bars
	}
	latest := bars[len(bars)-1].Time
	cutoff := latest
	if strings.HasSuffix(rangeValue, "y") {
		years, err := strconv.Atoi(strings.TrimSuffix(rangeValue, "y"))
		if err != nil || years <= 0 {
			return bars
		}
		cutoff = latest.AddDate(-years, 0, 0)
	} else if strings.HasSuffix(rangeValue, "mo") {
		months, err := strconv.Atoi(strings.TrimSuffix(rangeValue, "mo"))
		if err != nil || months <= 0 {
			return bars
		}
		cutoff = latest.AddDate(0, -months, 0)
	} else {
		return bars
	}
	start := 0
	for start < len(bars) && bars[start].Time.Before(cutoff) {
		start++
	}
	if start >= len(bars) {
		return bars
	}
	return bars[start:]
}

func (client *Client) doRequest(request *http.Request) ([]byte, error) {
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("%s %s: status=%d body=%s", request.Method, request.URL.String(), response.StatusCode, string(body))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func WriteReportArtifacts(root string, report core.Report) error {
	equityDir := filepath.Join(root, "equity")
	tradesDir := filepath.Join(root, "trades")
	if err := os.MkdirAll(equityDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(tradesDir, 0o755); err != nil {
		return err
	}
	prefix := sanitize(report.Name)
	if prefix == "" {
		prefix = sanitize(report.Symbol)
	}
	if err := writeEquityCSV(filepath.Join(equityDir, prefix+".csv"), report.EquityCurve); err != nil {
		return err
	}
	if err := writeTradesCSV(filepath.Join(tradesDir, prefix+".csv"), report.Trades); err != nil {
		return err
	}
	return nil
}

func datasetCachePath(root string, spec config.DatasetConfig) string {
	filename := fmt.Sprintf("%s_%s_%s.csv", sanitize(spec.Provider), sanitize(spec.Symbol), sanitize(spec.Interval))
	return filepath.Join(root, sanitize(spec.Provider), filename)
}

func sanitize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("/", "_", "=", "_", ":", "_", " ", "_", "-", "_")
	return replacer.Replace(value)
}

func writeBarsCSV(path string, bars []core.Bar) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"time", "open", "high", "low", "close", "volume"}); err != nil {
		return err
	}
	for _, bar := range bars {
		record := []string{bar.Time.UTC().Format(time.RFC3339), strconv.FormatFloat(bar.Open, 'f', -1, 64), strconv.FormatFloat(bar.High, 'f', -1, 64), strconv.FormatFloat(bar.Low, 'f', -1, 64), strconv.FormatFloat(bar.Close, 'f', -1, 64), strconv.FormatFloat(bar.Volume, 'f', -1, 64)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.Error()
}

func readBarsCSV(path string) ([]core.Bar, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	bars := make([]core.Bar, 0, len(records)-1)
	for index, record := range records {
		if index == 0 || len(record) < 6 {
			continue
		}
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, err
		}
		openValue, _ := strconv.ParseFloat(record[1], 64)
		highValue, _ := strconv.ParseFloat(record[2], 64)
		lowValue, _ := strconv.ParseFloat(record[3], 64)
		closeValue, _ := strconv.ParseFloat(record[4], 64)
		volumeValue, _ := strconv.ParseFloat(record[5], 64)
		bars = append(bars, core.Bar{Time: timestamp, Open: openValue, High: highValue, Low: lowValue, Close: closeValue, Volume: volumeValue})
	}
	return bars, nil
}

func writeEquityCSV(path string, curve []core.EquityPoint) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"time", "equity"}); err != nil {
		return err
	}
	for _, point := range curve {
		if err := writer.Write([]string{point.Time.UTC().Format(time.RFC3339), strconv.FormatFloat(point.Equity, 'f', -1, 64)}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func writeTradesCSV(path string, trades []core.Trade) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"side", "entry_time", "exit_time", "entry_price", "exit_price", "return", "bars_held", "reason", "score"}); err != nil {
		return err
	}
	for _, trade := range trades {
		record := []string{string(trade.Side), trade.EntryTime.UTC().Format(time.RFC3339), trade.ExitTime.UTC().Format(time.RFC3339), strconv.FormatFloat(trade.EntryPrice, 'f', -1, 64), strconv.FormatFloat(trade.ExitPrice, 'f', -1, 64), strconv.FormatFloat(trade.Return, 'f', -1, 64), strconv.Itoa(trade.BarsHeld), trade.Reason, strconv.FormatFloat(trade.Score, 'f', -1, 64)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.Error()
}

func bitgetGranularity(interval string) string {
	switch strings.ToLower(interval) {
	case "1m":
		return "1min"
	case "5m":
		return "5min"
	case "15m":
		return "15min"
	case "1h":
		return "1H"
	case "4h":
		return "4H"
	case "1d":
		return "1day"
	default:
		return "1day"
	}
}

func stooqInterval(interval string) string {
	switch strings.ToLower(interval) {
	case "1d":
		return "d"
	case "1w":
		return "w"
	case "1m":
		return "m"
	default:
		return "d"
	}
}

func asFloat64(value interface{}) (float64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseFloat(typed, 64)
	case float64:
		return typed, nil
	case int64:
		return float64(typed), nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}

func asInt64(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case float64:
		return int64(typed), nil
	case int64:
		return typed, nil
	default:
		return 0, fmt.Errorf("unsupported integer type %T", value)
	}
}
