package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func (client *Client) FetchContractRules(ctx context.Context, productType string) (map[string]ContractRule, error) {
	path := "/api/v2/mix/market/contracts?productType=" + url.QueryEscape(productType)
	body, err := client.doPublic(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	return decodeContractRules(body)
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
