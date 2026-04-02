package bitget

import "strings"

type ContractRule struct {
	Symbol         string
	MinTradeNum    float64
	SizeMultiplier float64
	MaxLeverage    int
}

type contractRulesResponse struct {
	Code string                `json:"code"`
	Msg  string                `json:"msg"`
	Data []contractRulePayload `json:"data"`
}

type contractRulePayload struct {
	Symbol         string `json:"symbol"`
	MinTradeNum    string `json:"minTradeNum"`
	SizeMultiplier string `json:"sizeMultiplier"`
	MaxLeverage    string `json:"maxLeverage"`
	MaxLever       string `json:"maxLever"`
}

func bitgetMarketType(productType string) string {
	if strings.Contains(strings.ToUpper(strings.TrimSpace(productType)), "SPOT") {
		return "spot"
	}
	return "perp"
}
