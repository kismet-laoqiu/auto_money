package adapters

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

func TestEnsureDatasetUsesCachedBarsWithoutFetch(t *testing.T) {
	cacheDir := t.TempDir()
	spec := config.DatasetConfig{
		Name:     "cached",
		Provider: "bitget",
		Symbol:   "MSTRUSDT",
		Interval: "1m",
	}
	bars := []core.Bar{{
		Time:   time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC),
		Open:   1,
		High:   2,
		Low:    0.5,
		Close:  1.5,
		Volume: 10,
	}}
	if err := writeBarsCSV(datasetCachePath(cacheDir, spec), bars); err != nil {
		t.Fatalf("write bars cache: %v", err)
	}
	client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("unexpected remote fetch")
	})}}
	dataset, err := client.EnsureDataset(context.Background(), cacheDir, spec, false)
	if err != nil {
		t.Fatalf("ensure dataset from cache: %v", err)
	}
	if len(dataset.Bars) != 1 || dataset.Bars[0].Close != 1.5 {
		t.Fatalf("unexpected dataset bars: %+v", dataset.Bars)
	}
}

func TestFetchBitgetUsesFuturesCandlesPathWhenProductTypeSet(t *testing.T) {
	var requestedURL string
	client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestedURL = req.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"code":"00000","msg":"success","data":[["1710000000000","1","2","0.5","1.5","10"]]}`)),
			Header:     make(http.Header),
		}, nil
	})}}
	_, err := client.fetchBitget(context.Background(), config.DatasetConfig{
		Provider:    "bitget",
		Symbol:      "MSTRUSDT",
		Interval:    "1m",
		Limit:       1,
		ProductType: "USDT-FUTURES",
	})
	if err != nil {
		t.Fatalf("fetch bitget futures bars: %v", err)
	}
	if !strings.Contains(requestedURL, "/api/v2/mix/market/candles") || !strings.Contains(requestedURL, "productType=USDT-FUTURES") {
		t.Fatalf("unexpected futures request url: %s", requestedURL)
	}
	parsedURL, err := url.Parse(requestedURL)
	if err != nil {
		t.Fatalf("parse requested url: %v", err)
	}
	if granularity := parsedURL.Query().Get("granularity"); granularity != "1m" {
		t.Fatalf("unexpected futures granularity %q in request url: %s", granularity, requestedURL)
	}
}

func TestBitgetMixEndpointClampsStartTimeForLongHistoryWindow(t *testing.T) {
	spec := config.DatasetConfig{
		Symbol:      "BTCUSDT",
		ProductType: "USDT-FUTURES",
		StartTime:   time.Date(2023, 3, 30, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
	}
	endpoint := bitgetMixEndpoint(spec, "15m", 500)
	if !strings.Contains(endpoint, "/api/v2/mix/market/history-candles") {
		t.Fatalf("expected history endpoint, got %s", endpoint)
	}
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("parse endpoint: %v", err)
	}
	expectedStart := strconv.FormatInt(spec.EndTime.Add(-90*24*time.Hour).UnixMilli(), 10)
	if got := parsedURL.Query().Get("startTime"); got != expectedStart {
		t.Fatalf("expected clamped startTime=%s, got %s in %s", expectedStart, got, endpoint)
	}
	if parsedURL.Query().Get("endTime") == "" {
		t.Fatalf("expected endTime in endpoint, got %s", endpoint)
	}
	if parsedURL.Query().Get("limit") != "200" {
		t.Fatalf("expected limit=200 in endpoint, got %s", endpoint)
	}
}

func TestFetchBitgetUsesSpotPathWithoutProductType(t *testing.T) {
	var requestedURL string
	client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestedURL = req.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"code":"00000","msg":"success","data":[["1710000000000","1","2","0.5","1.5","10"]]}`)),
			Header:     make(http.Header),
		}, nil
	})}}
	_, err := client.fetchBitget(context.Background(), config.DatasetConfig{
		Provider: "bitget",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("fetch bitget spot bars: %v", err)
	}
	if !strings.Contains(requestedURL, "/api/v2/spot/market/candles") {
		t.Fatalf("unexpected spot request url: %s", requestedURL)
	}
	parsedURL, err := url.Parse(requestedURL)
	if err != nil {
		t.Fatalf("parse requested url: %v", err)
	}
	if granularity := parsedURL.Query().Get("granularity"); granularity != "1min" {
		t.Fatalf("unexpected spot granularity %q in request url: %s", granularity, requestedURL)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
