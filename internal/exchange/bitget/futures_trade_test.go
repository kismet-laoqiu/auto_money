package bitget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDoPrivateSetsBitgetHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ACCESS-KEY") != "key" {
			t.Fatalf("missing api key header: %+v", r.Header)
		}
		if r.Header.Get("ACCESS-PASSPHRASE") != "pass" {
			t.Fatalf("missing passphrase header: %+v", r.Header)
		}
		if r.Header.Get("ACCESS-TIMESTAMP") == "" || r.Header.Get("ACCESS-SIGN") == "" {
			t.Fatalf("missing signed headers: %+v", r.Header)
		}
		if got := r.URL.String(); got != "/api/v2/mix/account/accounts?productType=USDT-FUTURES" {
			t.Fatalf("unexpected request path: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[]}`))
	}))
	defer server.Close()

	client := NewPrivateClient(server.URL, PrivateCredentials{
		Key:        "key",
		Secret:     "secret",
		Passphrase: "pass",
	})
	if _, err := client.doPrivate(context.Background(), http.MethodGet, "/api/v2/mix/account/accounts?productType=USDT-FUTURES", nil); err != nil {
		t.Fatalf("doPrivate: %v", err)
	}
}

func TestPrivateTradeSurfaceUsesExpectedPaths(t *testing.T) {
	cases := []struct {
		name string
		run  func(ctx context.Context, client *Client) error
		path string
		want map[string]string
	}{
		{
			name: "set leverage",
			path: "/api/v2/mix/account/set-leverage",
			want: map[string]string{
				"symbol":      "MSTRUSDT",
				"productType": "USDT-FUTURES",
				"marginCoin":  "USDT",
				"leverage":    "3",
				"holdSide":    "long",
			},
			run: func(ctx context.Context, client *Client) error {
				_, err := client.SetLeverage(ctx, SetLeverageRequest{
					Symbol:      "MSTRUSDT",
					ProductType: "USDT-FUTURES",
					MarginCoin:  "USDT",
					Leverage:    "3",
					HoldSide:    "long",
				})
				return err
			},
		},
		{
			name: "place order",
			path: "/api/v2/mix/order/place-order",
			want: map[string]string{
				"symbol":      "MSTRUSDT",
				"productType": "USDT-FUTURES",
				"marginMode":  "isolated",
				"marginCoin":  "USDT",
				"side":        "buy",
				"tradeSide":   "open",
				"orderType":   "market",
				"size":        "0.01",
				"clientOid":   "cid-1",
			},
			run: func(ctx context.Context, client *Client) error {
				_, err := client.PlaceOrder(ctx, PlaceOrderRequest{
					Symbol:      "MSTRUSDT",
					ProductType: "USDT-FUTURES",
					MarginMode:  "isolated",
					MarginCoin:  "USDT",
					Side:        "buy",
					TradeSide:   "open",
					OrderType:   "market",
					Size:        "0.01",
					ClientOID:   "cid-1",
				})
				return err
			},
		},
		{
			name: "get order detail",
			path: "/api/v2/mix/order/detail?symbol=MSTRUSDT&productType=USDT-FUTURES&orderId=oid-1",
			run: func(ctx context.Context, client *Client) error {
				_, err := client.GetOrderDetail(ctx, OrderDetailRequest{
					Symbol:      "MSTRUSDT",
					ProductType: "USDT-FUTURES",
					OrderID:     "oid-1",
				})
				return err
			},
		},
		{
			name: "cancel order",
			path: "/api/v2/mix/order/cancel-order",
			want: map[string]string{
				"symbol":      "MSTRUSDT",
				"productType": "USDT-FUTURES",
				"marginCoin":  "USDT",
				"orderId":     "oid-1",
			},
			run: func(ctx context.Context, client *Client) error {
				_, err := client.CancelOrder(ctx, CancelOrderRequest{
					Symbol:      "MSTRUSDT",
					ProductType: "USDT-FUTURES",
					MarginCoin:  "USDT",
					OrderID:     "oid-1",
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expected, err := url.Parse(tc.path)
				if err != nil {
					t.Fatalf("parse expected path: %v", err)
				}
				if r.URL.Path != expected.Path {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.URL.RawQuery != expected.RawQuery {
					gotValues, err := url.ParseQuery(r.URL.RawQuery)
					if err != nil {
						t.Fatalf("parse actual query: %v", err)
					}
					wantValues, err := url.ParseQuery(expected.RawQuery)
					if err != nil {
						t.Fatalf("parse expected query: %v", err)
					}
					if gotValues.Encode() != wantValues.Encode() {
						t.Fatalf("unexpected query: got=%s want=%s", gotValues.Encode(), wantValues.Encode())
					}
				}
				if len(tc.want) != 0 {
					var payload map[string]string
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Fatalf("decode body: %v", err)
					}
					for key, value := range tc.want {
						if payload[key] != value {
							t.Fatalf("unexpected body[%s]=%q want %q body=%v", key, payload[key], value, payload)
						}
					}
				}
				w.Header().Set("Content-Type", "application/json")
				switch {
				case strings.Contains(r.URL.Path, "detail"):
					_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"orderId":"oid-1","clientOid":"cid-1","state":"filled","priceAvg":"100","size":"0.01","reduceOnly":"NO"}}`))
				case strings.Contains(r.URL.Path, "set-leverage"):
					_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"symbol":"MSTRUSDT","marginCoin":"USDT","longLeverage":"3","shortLeverage":"3","crossMarginLeverage":"3","marginMode":"isolated"}}`))
				default:
					_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":{"orderId":"oid-1","clientOid":"cid-1"}}`))
				}
			}))
			defer server.Close()

			client := NewPrivateClient(server.URL, PrivateCredentials{
				Key:        "key",
				Secret:     "secret",
				Passphrase: "pass",
			})
			if err := tc.run(context.Background(), client); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
		})
	}
}

func TestFetchPrivateSnapshotsUsesExpectedPaths(t *testing.T) {
	cases := []struct {
		name string
		run  func(ctx context.Context, client *Client) error
		path string
	}{
		{
			name: "fetch futures account",
			path: "/api/v2/mix/account/accounts?productType=USDT-FUTURES",
			run: func(ctx context.Context, client *Client) error {
				_, err := client.FetchFuturesAccounts(ctx, "USDT-FUTURES")
				return err
			},
		},
		{
			name: "fetch futures positions",
			path: "/api/v2/mix/position/all-position?productType=USDT-FUTURES&marginCoin=USDT",
			run: func(ctx context.Context, client *Client) error {
				_, err := client.FetchFuturesPositions(ctx, "USDT-FUTURES", "USDT")
				return err
			},
		},
		{
			name: "fetch single position",
			path: "/api/v2/mix/position/single-position?symbol=MSTRUSDT&productType=USDT-FUTURES&marginCoin=USDT",
			run: func(ctx context.Context, client *Client) error {
				_, err := client.FetchSinglePosition(ctx, "MSTRUSDT", "USDT-FUTURES", "USDT")
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expected, err := url.Parse(tc.path)
				if err != nil {
					t.Fatalf("parse expected path: %v", err)
				}
				if r.URL.Path != expected.Path {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.URL.Query().Encode() != expected.Query().Encode() {
					t.Fatalf("unexpected query: got=%s want=%s", r.URL.Query().Encode(), expected.Query().Encode())
				}
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "accounts") {
					_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDT","available":"10","equity":"12","usdtEquity":"12","unrealizedPL":"0"}]}`))
					return
				}
				if strings.Contains(r.URL.Path, "single-position") {
					_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"holdSide":"long","total":"0.04","posMode":"one_way_mode"}]}`))
					return
				}
				_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"MSTRUSDT","holdSide":"long","total":"0.01","uTime":"1710000000000"}]}`))
			}))
			defer server.Close()

			client := NewPrivateClient(server.URL, PrivateCredentials{
				Key:        "key",
				Secret:     "secret",
				Passphrase: "pass",
			})
			if err := tc.run(context.Background(), client); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
		})
	}
}

func TestFetchSinglePositionDecodesSignedQtyAndMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected, err := url.Parse("/api/v2/mix/position/single-position?symbol=MSTRUSDT&productType=USDT-FUTURES&marginCoin=USDT")
		if err != nil {
			t.Fatalf("parse expected path: %v", err)
		}
		if r.URL.Path != expected.Path {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Encode() != expected.Query().Encode() {
			t.Fatalf("unexpected query: got=%s want=%s", r.URL.Query().Encode(), expected.Query().Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"holdSide":"short","total":"0.04","posMode":"hedge_mode"}]}`))
	}))
	defer server.Close()

	client := NewPrivateClient(server.URL, PrivateCredentials{
		Key:        "key",
		Secret:     "secret",
		Passphrase: "pass",
	})
	got, err := client.FetchSinglePosition(context.Background(), "MSTRUSDT", "USDT-FUTURES", "USDT")
	if err != nil {
		t.Fatalf("fetch single position: %v", err)
	}
	if got.Qty != -0.04 || got.Mode != "hedge_mode" {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestFetchSinglePositionEmptyDataReturnsFlatSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected, err := url.Parse("/api/v2/mix/position/single-position?symbol=MSTRUSDT&productType=USDT-FUTURES&marginCoin=USDT")
		if err != nil {
			t.Fatalf("parse expected path: %v", err)
		}
		if r.URL.Path != expected.Path {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Encode() != expected.Query().Encode() {
			t.Fatalf("unexpected query: got=%s want=%s", r.URL.Query().Encode(), expected.Query().Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[]}`))
	}))
	defer server.Close()

	client := NewPrivateClient(server.URL, PrivateCredentials{
		Key:        "key",
		Secret:     "secret",
		Passphrase: "pass",
	})
	got, err := client.FetchSinglePosition(context.Background(), "MSTRUSDT", "USDT-FUTURES", "USDT")
	if err != nil {
		t.Fatalf("fetch single position: %v", err)
	}
	if got.Qty != 0 || got.Mode != "" {
		t.Fatalf("unexpected empty-position snapshot: %+v", got)
	}
}

func TestPrivateAccountSnapshotAndStreamEventIDsDoNotCollide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/api/v2/mix/account/accounts?productType=USDT-FUTURES" {
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"marginCoin":"USDT","available":"10","equity":"12","usdtEquity":"12","unrealizedPL":"0"}]}`))
	}))
	defer server.Close()

	client := NewPrivateClient(server.URL, PrivateCredentials{
		Key:        "key",
		Secret:     "secret",
		Passphrase: "pass",
	})
	accounts, err := client.FetchFuturesAccounts(context.Background(), "USDT-FUTURES")
	if err != nil {
		t.Fatalf("fetch futures accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected one account snapshot, got %d", len(accounts))
	}
	streamEvents, err := DecodePrivateEvents([]byte(`{"arg":{"channel":"account","coin":"default"},"data":[{"marginCoin":"USDT","available":"10","equity":"12","usdtEquity":"12","unrealizedPL":"0"}]}`))
	if err != nil {
		t.Fatalf("decode private account stream: %v", err)
	}
	if len(streamEvents) != 1 {
		t.Fatalf("expected one stream account event, got %d", len(streamEvents))
	}
	if accounts[0].EventID() == streamEvents[0].EventID() {
		t.Fatalf("snapshot and stream account event ids collided: %s", accounts[0].EventID())
	}
}

func TestPrivatePositionSnapshotAndStreamEventIDsDoNotCollide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/api/v2/mix/position/all-position?productType=USDT-FUTURES&marginCoin=USDT" {
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"00000","msg":"success","data":[{"symbol":"MSTRUSDT","holdSide":"long","total":"0.01","uTime":"1710000000000"}]}`))
	}))
	defer server.Close()

	client := NewPrivateClient(server.URL, PrivateCredentials{
		Key:        "key",
		Secret:     "secret",
		Passphrase: "pass",
	})
	positions, err := client.FetchFuturesPositions(context.Background(), "USDT-FUTURES", "USDT")
	if err != nil {
		t.Fatalf("fetch futures positions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected one position snapshot, got %d", len(positions))
	}
	streamEvents, err := DecodePrivateEvents([]byte(`{"arg":{"channel":"positions","instId":"default"},"data":[{"instId":"MSTRUSDT","holdSide":"long","total":"0.01","uTime":"1710000000000"}]}`))
	if err != nil {
		t.Fatalf("decode private position stream: %v", err)
	}
	if len(streamEvents) != 1 {
		t.Fatalf("expected one stream position event, got %d", len(streamEvents))
	}
	if positions[0].EventID() == streamEvents[0].EventID() {
		t.Fatalf("snapshot and stream position event ids collided: %s", positions[0].EventID())
	}
}
