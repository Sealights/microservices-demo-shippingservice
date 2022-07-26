package main

type CartItem struct {
	ProductId string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type Address struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       int32  `json:"zip_code"`
}

type ShipOrderRequest struct {
	Address *Address    `json:"address"`
	Items   []*CartItem `json:"cart_item"`
}

type QuoteResponse struct {
	CostUsd *Money `json:"cost_usd"`
}

type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type ShipOrderResponse struct {
	TrackingId string `json:"tracking_id"`
}
