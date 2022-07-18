package main

type CartItem struct {
	ProductId string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}

type Address struct {
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       int32  `json:"zipCode"`
}

type ShipOrderRequest struct {
	Address *Address    `json:"address"`
	Items   []*CartItem `json:"cartItem"`
}

type QuoteResponse struct {
	CostUsd *Money `json:"costUsd"`
}

type Money struct {
	CurrencyCode string `json:"currencyCode"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type ShipOrderResponse struct {
	TrackingId string `json:"trackingId"`
}
