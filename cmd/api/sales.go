package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/obynonwane/inventory-service/data"
)

type CreatePrurchaseOrderPayload struct {
	InventoryId       string  `json:"inventory_id" binding:"required"`
	SellerId          string  `json:"seller_id"`
	BuyerId           string  `json:"buyer_id"`
	OfferPricePerUnit float64 `json:"offer_price_per_unit" binding:"required"`
	Quantity          float64 `json:"quantity" binding:"required"`
	TotalAmount       float64 `json:"total_amount" binding:"required"`
}

func (app *Config) CreatePrurchaseOrder(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload CreatePrurchaseOrderPayload
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// get the inventory
	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()
	inv, err := app.Repo.GetInventoryByID(timeoutCtx, requestPayload.InventoryId)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}
	// check if item is for sale
	if inv.ProductPurpose != "sale" {
		app.errorJSON(w, errors.New(fmt.Sprintf("item is only for outright purchase not rental")), nil, http.StatusBadRequest)
		return
	}

	// check check the offer price is not less than stipulated price
	if requestPayload.OfferPricePerUnit < inv.MinimumPrice {
		app.errorJSON(w, errors.New(fmt.Sprintf("offer price can not be less than minimum price: %v", inv.MinimumPrice)), nil, http.StatusBadRequest)
		return
	}

	// check the offer price is not more than stipulated price
	if requestPayload.OfferPricePerUnit > inv.OfferPrice {
		app.errorJSON(w, errors.New(fmt.Sprintf("offer price can not be more than stipulated price: %v", inv.OfferPrice)), nil, http.StatusBadRequest)
		return
	}
	// check the quantity needed is met
	if requestPayload.Quantity > inv.Quantity {
		app.errorJSON(w, errors.New(fmt.Sprintf("the stipulated quantity is not available, only: %v", inv.Quantity)), nil, http.StatusBadRequest)
		return
	}

	// calculate the total amount (quantity * offer_price_per_unit)
	totalPrice := float64(requestPayload.Quantity) * requestPayload.OfferPricePerUnit

	bookings, err := app.Repo.CreatePurchaseOrder(timeoutCtx, &data.CreatePurchaseOrderPayload{
		SellerId:          inv.UserId,
		BuyerId:           requestPayload.BuyerId,
		InventoryId:       inv.ID,
		OfferPricePerUnit: requestPayload.OfferPricePerUnit,
		Quantity:          int32(requestPayload.Quantity),
		TotalAmount:       totalPrice,
	})
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	// send sms & email notification to both owner and buyer

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "purchase order created successfully",
		Data:       bookings,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
