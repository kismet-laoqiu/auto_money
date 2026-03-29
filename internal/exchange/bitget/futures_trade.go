package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"quantlab/internal/market"
)

type PlaceOrderRequest struct {
	Symbol      string `json:"symbol"`
	ProductType string `json:"productType"`
	MarginMode  string `json:"marginMode"`
	MarginCoin  string `json:"marginCoin,omitempty"`
	Side        string `json:"side"`
	TradeSide   string `json:"tradeSide,omitempty"`
	OrderType   string `json:"orderType"`
	Size        string `json:"size"`
	Price       string `json:"price,omitempty"`
	ReduceOnly  string `json:"reduceOnly,omitempty"`
	ClientOID   string `json:"clientOid,omitempty"`
}

type SetLeverageRequest struct {
	Symbol      string `json:"symbol"`
	ProductType string `json:"productType"`
	MarginCoin  string `json:"marginCoin"`
	Leverage    string `json:"leverage"`
	HoldSide    string `json:"holdSide,omitempty"`
}

type CancelOrderRequest struct {
	Symbol      string `json:"symbol"`
	ProductType string `json:"productType"`
	MarginCoin  string `json:"marginCoin,omitempty"`
	OrderID     string `json:"orderId,omitempty"`
	ClientOID   string `json:"clientOid,omitempty"`
}

type OrderDetailRequest struct {
	Symbol      string
	ProductType string
	OrderID     string
	ClientOID   string
}

type OrderResult struct {
	OrderID   string `json:"orderId"`
	ClientOID string `json:"clientOid"`
}

type OrderDetail struct {
	OrderID    string
	ClientOID  string
	Status     string
	Size       float64
	PriceAvg   float64
	ReduceOnly bool
}

type LeverageSetting struct {
	Symbol              string `json:"symbol"`
	MarginCoin          string `json:"marginCoin"`
	LongLeverage        string `json:"longLeverage"`
	ShortLeverage       string `json:"shortLeverage"`
	CrossMarginLeverage string `json:"crossMarginLeverage"`
	MarginMode          string `json:"marginMode"`
}

func (client *Client) FetchFuturesAccounts(ctx context.Context, productType string) ([]market.AccountEvent, error) {
	path := "/api/v2/mix/account/accounts?productType=" + url.QueryEscape(productType)
	body, err := client.doPrivate(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			MarginCoin   string `json:"marginCoin"`
			Available    string `json:"available"`
			Equity       string `json:"equity"`
			AccountEq    string `json:"accountEquity"`
			USDTEq       string `json:"usdtEquity"`
			UnrealizedPL string `json:"unrealizedPL"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode futures accounts: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return nil, fmt.Errorf("bitget futures accounts code=%s msg=%s", response.Code, response.Msg)
	}
	events := make([]market.AccountEvent, 0, len(response.Data))
	now := time.Now().UTC()
	for index, row := range response.Data {
		equityRaw := row.Equity
		if equityRaw == "" {
			equityRaw = row.AccountEq
		}
		available, err := strconv.ParseFloat(row.Available, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account available: %w", err)
		}
		equity, err := strconv.ParseFloat(equityRaw, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account equity: %w", err)
		}
		usdtEq, err := strconv.ParseFloat(row.USDTEq, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account usdt equity: %w", err)
		}
		unrealizedPL, err := strconv.ParseFloat(row.UnrealizedPL, 64)
		if err != nil {
			return nil, fmt.Errorf("parse account unrealized pnl: %w", err)
		}
		events = append(events, market.AccountEvent{
			EventIDValue: fmt.Sprintf("account:%s:%d", row.MarginCoin, index),
			SymbolValue:  row.MarginCoin,
			Ts:           now,
			MarginCoin:   row.MarginCoin,
			Available:    available,
			Equity:       equity,
			USDTEq:       usdtEq,
			UnrealizedPL: unrealizedPL,
		})
	}
	return events, nil
}

func (client *Client) FetchFuturesPositions(ctx context.Context, productType, marginCoin string) ([]market.PositionEvent, error) {
	path := "/api/v2/mix/position/all-position?productType=" + url.QueryEscape(productType)
	if marginCoin != "" {
		path += "&marginCoin=" + url.QueryEscape(marginCoin)
	}
	body, err := client.doPrivate(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Symbol   string `json:"symbol"`
			HoldSide string `json:"holdSide"`
			Total    string `json:"total"`
			UTime    string `json:"uTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode futures positions: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return nil, fmt.Errorf("bitget futures positions code=%s msg=%s", response.Code, response.Msg)
	}
	events := make([]market.PositionEvent, 0, len(response.Data))
	for index, row := range response.Data {
		qty, err := parseSignedPositionQty(row.HoldSide, row.Total)
		if err != nil {
			return nil, err
		}
		ts, err := parseMillis(row.UTime)
		if err != nil {
			return nil, err
		}
		events = append(events, market.PositionEvent{
			EventIDValue: fmt.Sprintf("position:%s:%d", row.Symbol, index),
			SymbolValue:  row.Symbol,
			Ts:           ts,
			Qty:          qty,
		})
	}
	return events, nil
}

func (client *Client) SetLeverage(ctx context.Context, req SetLeverageRequest) (LeverageSetting, error) {
	body, err := client.doPrivate(ctx, http.MethodPost, "/api/v2/mix/account/set-leverage", req)
	if err != nil {
		return LeverageSetting{}, err
	}
	var response struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data LeverageSetting `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return LeverageSetting{}, fmt.Errorf("decode set leverage: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return LeverageSetting{}, fmt.Errorf("bitget set leverage code=%s msg=%s", response.Code, response.Msg)
	}
	return response.Data, nil
}

func (client *Client) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (OrderResult, error) {
	body, err := client.doPrivate(ctx, http.MethodPost, "/api/v2/mix/order/place-order", req)
	if err != nil {
		return OrderResult{}, err
	}
	return decodeOrderResult(body, "place order")
}

func (client *Client) GetOrderDetail(ctx context.Context, req OrderDetailRequest) (OrderDetail, error) {
	values := url.Values{}
	values.Set("symbol", req.Symbol)
	values.Set("productType", req.ProductType)
	if req.OrderID != "" {
		values.Set("orderId", req.OrderID)
	}
	if req.ClientOID != "" {
		values.Set("clientOid", req.ClientOID)
	}
	body, err := client.doPrivate(ctx, http.MethodGet, "/api/v2/mix/order/detail?"+values.Encode(), nil)
	if err != nil {
		return OrderDetail{}, err
	}
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderID    string `json:"orderId"`
			ClientOID  string `json:"clientOid"`
			State      string `json:"state"`
			Status     string `json:"status"`
			Size       string `json:"size"`
			PriceAvg   string `json:"priceAvg"`
			ReduceOnly string `json:"reduceOnly"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return OrderDetail{}, fmt.Errorf("decode order detail: %w", err)
	}
	if response.Code != "" && response.Code != "00000" {
		return OrderDetail{}, fmt.Errorf("bitget order detail code=%s msg=%s", response.Code, response.Msg)
	}
	size, err := strconv.ParseFloat(response.Data.Size, 64)
	if err != nil {
		return OrderDetail{}, fmt.Errorf("parse order size: %w", err)
	}
	priceAvg, err := strconv.ParseFloat(response.Data.PriceAvg, 64)
	if err != nil {
		return OrderDetail{}, fmt.Errorf("parse order avg price: %w", err)
	}
	status := response.Data.Status
	if status == "" {
		status = response.Data.State
	}
	return OrderDetail{
		OrderID:    response.Data.OrderID,
		ClientOID:  response.Data.ClientOID,
		Status:     status,
		Size:       size,
		PriceAvg:   priceAvg,
		ReduceOnly: strings.EqualFold(response.Data.ReduceOnly, "yes"),
	}, nil
}

func (client *Client) CancelOrder(ctx context.Context, req CancelOrderRequest) (OrderResult, error) {
	body, err := client.doPrivate(ctx, http.MethodPost, "/api/v2/mix/order/cancel-order", req)
	if err != nil {
		return OrderResult{}, err
	}
	return decodeOrderResult(body, "cancel order")
}

func decodeOrderResult(body []byte, action string) (OrderResult, error) {
	var response struct {
		Code string      `json:"code"`
		Msg  string      `json:"msg"`
		Data OrderResult `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResult{}, fmt.Errorf("decode %s: %w", action, err)
	}
	if response.Code != "" && response.Code != "00000" {
		return OrderResult{}, fmt.Errorf("bitget %s code=%s msg=%s", action, response.Code, response.Msg)
	}
	return response.Data, nil
}

func parseSignedPositionQty(holdSide, total string) (float64, error) {
	qty, err := strconv.ParseFloat(total, 64)
	if err != nil {
		return 0, fmt.Errorf("parse position qty: %w", err)
	}
	if holdSide == "short" {
		return -qty, nil
	}
	return qty, nil
}

func parseMillis(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse millis %q: %w", value, err)
	}
	return time.UnixMilli(millis).UTC(), nil
}
