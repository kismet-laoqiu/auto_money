package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"quantlab/internal/market"
)

func (client *Client) FetchContractRules(ctx context.Context, productType string) (map[string]ContractRule, error) {
	path := "/api/v2/mix/market/contracts?productType=" + url.QueryEscape(productType)
	body, err := client.doPublic(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	return decodeContractRules(body)
}

func (client *Client) FetchCandles(ctx context.Context, symbol, productType, interval string, limit int) ([]market.BarClosedEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	path := fmt.Sprintf(
		"/api/v2/mix/market/candles?symbol=%s&productType=%s&granularity=%s&limit=%d",
		url.QueryEscape(symbol),
		url.QueryEscape(productType),
		url.QueryEscape(candleGranularity(interval)),
		limit,
	)
	body, err := client.doPublic(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	return decodeCandleEvents(symbol, productType, interval, body)
}

func (client *Client) FetchTickerPrice(ctx context.Context, symbol, productType string) (float64, error) {
	path := fmt.Sprintf(
		"/api/v2/mix/market/ticker?symbol=%s&productType=%s",
		url.QueryEscape(symbol),
		url.QueryEscape(productType),
	)
	body, err := client.doPublic(ctx, http.MethodGet, path)
	if err != nil {
		return 0, err
	}
	return decodeTickerPrice(body)
}

func decodeTickerPrice(body []byte) (float64, error) {
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			LastPrice string `json:"lastPr"`
			MarkPrice string `json:"markPrice"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("decode ticker response: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return 0, fmt.Errorf("bitget ticker code=%s msg=%s", response.Code, response.Msg)
	}
	if len(response.Data) == 0 {
		return 0, fmt.Errorf("bitget ticker returned empty data")
	}
	for _, raw := range []string{response.Data[0].LastPrice, response.Data[0].MarkPrice} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		price, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, fmt.Errorf("parse ticker price: %w", err)
		}
		if price > 0 {
			return price, nil
		}
	}
	return 0, fmt.Errorf("bitget ticker price missing for first row")
}

func decodeContractRules(body []byte) (map[string]ContractRule, error) {
	var response contractRulesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode contract rules: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return nil, fmt.Errorf("bitget contract rules code=%s msg=%s", response.Code, response.Msg)
	}
	rules := make(map[string]ContractRule, len(response.Data))
	for _, item := range response.Data {
		minTradeNum, err := strconv.ParseFloat(item.MinTradeNum, 64)
		if err != nil {
			return nil, fmt.Errorf("parse minTradeNum for %s: %w", item.Symbol, err)
		}
		sizeMultiplier, err := strconv.ParseFloat(item.SizeMultiplier, 64)
		if err != nil {
			return nil, fmt.Errorf("parse sizeMultiplier for %s: %w", item.Symbol, err)
		}
		maxLeverageValue := item.MaxLeverage
		if maxLeverageValue == "" {
			maxLeverageValue = item.MaxLever
		}
		maxLeverage, err := strconv.Atoi(maxLeverageValue)
		if err != nil {
			return nil, fmt.Errorf("parse maxLeverage for %s: %w", item.Symbol, err)
		}
		rules[item.Symbol] = ContractRule{
			Symbol:         item.Symbol,
			MinTradeNum:    minTradeNum,
			SizeMultiplier: sizeMultiplier,
			MaxLeverage:    maxLeverage,
		}
	}
	return rules, nil
}

func decodeCandleEvents(symbol, productType, interval string, body []byte) ([]market.BarClosedEvent, error) {
	var response struct {
		Code string     `json:"code"`
		Msg  string     `json:"msg"`
		Data [][]string `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode candle response: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return nil, fmt.Errorf("bitget candles code=%s msg=%s", response.Code, response.Msg)
	}
	events := make([]market.BarClosedEvent, 0, len(response.Data))
	for _, row := range response.Data {
		if len(row) < 6 {
			continue
		}
		ts, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle ts: %w", err)
		}
		openValue, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle open: %w", err)
		}
		highValue, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle high: %w", err)
		}
		lowValue, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle low: %w", err)
		}
		closeValue, err := strconv.ParseFloat(row[4], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle close: %w", err)
		}
		volumeValue, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			return nil, fmt.Errorf("parse candle volume: %w", err)
		}
		events = append(events, market.BarClosedEvent{
			EventIDValue: fmt.Sprintf("bootstrap:%s:%s:%d", symbol, interval, ts),
			SymbolValue:  symbol,
			Venue:        "bitget",
			MarketType:   bitgetMarketType(productType),
			Interval:     interval,
			Ts:           time.UnixMilli(ts).UTC(),
			Open:         openValue,
			High:         highValue,
			Low:          lowValue,
			Close:        closeValue,
			Volume:       volumeValue,
		})
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Ts.Before(events[j].Ts)
	})
	return events, nil
}

func CandleChannel(interval string) string {
	return "candle" + candleGranularity(interval)
}

func candleGranularity(interval string) string {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "1h":
		return "1H"
	case "4h":
		return "4H"
	case "1d":
		return "1D"
	case "1w":
		return "1W"
	default:
		return strings.TrimSpace(interval)
	}
}
