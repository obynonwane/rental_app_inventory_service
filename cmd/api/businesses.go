package main

import (
	"context"
	"net/http"
	"time"
)

func (app *Config) PremiumPartner(w http.ResponseWriter, r *http.Request) {

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	data, err := app.Repo.GetPremiumPartners(timeoutCtx)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	// send sms & email notification to both owner and buyer
	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
