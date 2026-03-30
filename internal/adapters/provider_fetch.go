package adapters

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	granularity := bitgetSpotGranularity(spec.Interval)
	endpoint := fmt.Sprintf("https://api.bitget.com/api/v2/spot/market/candles?symbol=%s&granularity=%s&limit=%d", url.QueryEscape(strings.ToUpper(spec.Symbol)), url.QueryEscape(granularity), limit)
	if spec.ProductType != "" {
		granularity = bitgetMixGranularity(spec.Interval)
		endpoint = bitgetMixEndpoint(spec, granularity, limit)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	body, err := client.doRequest(request)
	if err != nil {
		return nil, err
	}
	return decodeBitgetBars(spec.Symbol, body)
}

func bitgetMixEndpoint(spec config.DatasetConfig, granularity string, limit int) string {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(spec.Symbol))
	params.Set("productType", spec.ProductType)
	params.Set("granularity", granularity)
	if !spec.StartTime.IsZero() || !spec.EndTime.IsZero() {
		if limit <= 0 || limit > 200 {
			limit = 200
		}
		params.Set("limit", strconv.Itoa(limit))
		if !spec.StartTime.IsZero() && (spec.EndTime.IsZero() || spec.EndTime.Sub(spec.StartTime) <= 90*24*time.Hour) {
			params.Set("startTime", strconv.FormatInt(spec.StartTime.UTC().UnixMilli(), 10))
		}
		if !spec.EndTime.IsZero() {
			params.Set("endTime", strconv.FormatInt(spec.EndTime.UTC().UnixMilli(), 10))
		}
		return "https://api.bitget.com/api/v2/mix/market/history-candles?" + params.Encode()
	}
	params.Set("limit", strconv.Itoa(limit))
	return "https://api.bitget.com/api/v2/mix/market/candles?" + params.Encode()
}

func decodeBitgetBars(symbol string, body []byte) ([]core.Bar, error) {
	var payload struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data [][]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode bitget %s: %w", symbol, err)
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

func bitgetSpotGranularity(interval string) string {
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

func bitgetMixGranularity(interval string) string {
	switch strings.ToLower(interval) {
	case "1m":
		return "1m"
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "1h":
		return "1H"
	case "4h":
		return "4H"
	case "1d":
		return "1D"
	case "1w":
		return "1W"
	default:
		return "1D"
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
