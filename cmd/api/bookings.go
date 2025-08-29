package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/obynonwane/inventory-service/data"
	"github.com/obynonwane/inventory-service/utility"
)

type CreateBookingPayload struct {
	InventoryId       string  `json:"inventory_id" binding:"required"`
	RenterId          string  `json:"renter_id"`
	OwnerId           string  `json:"owner_id"`
	RentalType        string  `json:"rental_type" binding:"required"`     // e.g., "hourly", "daily"
	RentalDuration    float64 `json:"rental_duration" binding:"required"` // number of units (hours, days, etc.)
	SecurityDeposit   float64 `json:"security_deposit"`                   // can be zero
	OfferPricePerUnit float64 `json:"offer_price_per_unit" binding:"required"`
	Quantity          float64 `json:"quantity" binding:"required"`

	StartDate string `json:"start_date" binding:"required"` // e.g., "2025-06-15"
	EndDate   string `json:"end_date" binding:"required"`   // e.g., "2025-06-15"
	EndTime   string `json:"end_time" binding:"required"`   // e.g., "18:00:00", optional for daily+ rentals
	StartTime string `json:"start_time" binding:"required"` // e.g., "18:00:00", optional for daily+ rentals

	TotalAmount float64 `json:"total_amount" binding:"required"`
}

func (app *Config) CreateBooking(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload CreateBookingPayload
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
	if inv.ProductPurpose != "rental" {
		app.errorJSON(w, errors.New(fmt.Sprintf("item is only for rental not for sale")), nil, http.StatusBadRequest)
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
		app.errorJSON(w, errors.New(fmt.Sprintf("the stipulated quantity is not available, only: %v is available", inv.Quantity)), nil, http.StatusBadRequest)
		return
	}
	// check that the inventory is for the rental type
	if requestPayload.RentalType != inv.RentalDuration {
		app.errorJSON(w, errors.New(fmt.Sprintf("error: the rental type for this item is: %v", inv.RentalDuration)), nil, http.StatusBadRequest)
		return
	}

	// format startDate, endDate and endTime
	layout := "2006-01-02" // for date in format YYYY-MM-DD

	startDate, err := time.Parse(layout, requestPayload.StartDate)
	if err != nil {
		app.errorJSON(w, fmt.Errorf("invalid start date format, use YYYY-MM-DD"), nil, http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse(layout, requestPayload.EndDate)
	if err != nil {
		app.errorJSON(w, fmt.Errorf("invalid end date format, use YYYY-MM-DD"), nil, http.StatusBadRequest)
		return
	}

	timeLayout := "15:04" // for time in HH:MM

	_, err = time.Parse(timeLayout, requestPayload.EndTime)
	if err != nil {
		app.errorJSON(w, fmt.Errorf("invalid end time format, use HH:MM (24-hour format)"), nil, http.StatusBadRequest)
		return
	}

	_, err = time.Parse(timeLayout, requestPayload.StartTime)
	if err != nil {
		app.errorJSON(w, fmt.Errorf("invalid start time format, use HH:MM (24-hour format)"), nil, http.StatusBadRequest)
		return
	}

	// making sure the end date and start date is not in the past
	err = utility.ValidateBookingDates(startDate, endDate)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusBadRequest)
		return
	}

	// calculate the total amount (quantity * offer_price_per_unit)
	totalPrice := float64(requestPayload.Quantity) * requestPayload.OfferPricePerUnit * float64(requestPayload.RentalDuration)

	bookings, err := app.Repo.CreateBooking(timeoutCtx, &data.CreateBookingPayload{
		OwnerId:           inv.UserId,
		RenterId:          requestPayload.RenterId,
		InventoryId:       inv.ID,
		RentalType:        requestPayload.RentalType,
		RentalDuration:    int32(requestPayload.RentalDuration),
		SecurityDeposit:   requestPayload.SecurityDeposit,
		OfferPricePerUnit: requestPayload.OfferPricePerUnit,
		Quantity:          int32(requestPayload.Quantity),
		TotalAmount:       totalPrice,
		StartDate:         startDate,
		EndDate:           endDate,
		EndTime:           requestPayload.EndTime,
		StartTime:         requestPayload.StartTime,
	})
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	// send sms & email notification to both owner and renter

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "booking created successfully",
		Data:       bookings,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) MyBookings(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.MyBookingPayload
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Printf("%v", err)
		app.errorJSON(w, err, nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	bookings, err := app.Repo.GetMyBookings(timeoutCtx, requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	var payload = jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "booking retrieved successfully",
		Data:       bookings,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) GetBookingRequest(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.MyBookingPayload
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Printf("%v", err)
		app.errorJSON(w, err, nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	bookings, err := app.Repo.GetBookingRequest(timeoutCtx, requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}
	var payload = jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "booking retrieved successfully",
		Data:       bookings,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
