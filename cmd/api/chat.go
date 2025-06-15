package main

import (
	"context"
	"net/http"
	"time"

	"github.com/obynonwane/inventory-service/data"
)

func (app *Config) SubmitChat(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.Message
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

	chat, err := app.Repo.SubmitChat(timeoutCtx, &data.Message{
		Content:  requestPayload.Content,
		Sender:   requestPayload.Sender,
		Receiver: requestPayload.Receiver,
		SentAt:   requestPayload.SentAt,
	})
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	// send sms & email notification to both owner and buyer

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "chat created successfully",
		Data:       chat,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
