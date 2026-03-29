package adapters

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"quantlab/internal/config"
)

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
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
