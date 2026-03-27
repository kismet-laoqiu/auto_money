package bitget

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
