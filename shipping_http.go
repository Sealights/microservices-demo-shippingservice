package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func GetQuote(w http.ResponseWriter, r *http.Request) {
	log.Info("[GetQuote] received request")
	defer log.Info("[GetQuote] completed request")

	// Declare a new Person struct.
	var sor ShipOrderRequest

	err := json.NewDecoder(r.Body).Decode(&sor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	count := 0
	for _, item := range sor.Items {
		count += int(item.Quantity)
	}

	quote := CreateQuoteFromCount(count)

	var qRes QuoteResponse
	qRes.CostUsd = &Money{
		CurrencyCode: "USD",
		Units:        int64(quote.Dollars),
		Nanos:        int32(quote.Cents * 10000000)}

	json, err := json.Marshal(qRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Write(json)
}

func ShipOrder(w http.ResponseWriter, r *http.Request) {
	log.Info("[ShipOrder] received request")
	defer log.Info("[ShipOrder] completed request")

	var sor ShipOrderRequest

	err := json.NewDecoder(r.Body).Decode(&sor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	baseAddress := fmt.Sprintf("%s, %s, %s", sor.Address.StreetAddress, sor.Address.City, sor.Address.State)
	id := CreateTrackingId(baseAddress)

	var qRes ShipOrderResponse
	qRes.TrackingId = id

	json, err := json.Marshal(qRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Write(json)
}
