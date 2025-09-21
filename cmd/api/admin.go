package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/obynonwane/inventory-service/data"
)

func (app *Config) AdminGetInventoryPendingApproval(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.AdminPendingInventoryPayload
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

	// get the user rating and count
	data, err := app.Repo.GetAdminGetInventoryPending(timeoutCtx, requestPayload)
	if err != nil {
		log.Fatal("error retrieving user rating: %w", err)
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved succesfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)

}

func (app *Config) AdminApproveInventory(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	if id == "" {
		app.errorJSON(w, errors.New("id parameter is missing"), nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	// get the user rating and count
	err := app.Repo.AdminApproveInventory(timeoutCtx, id)
	if err != nil {
		log.Fatal("error approving inventory: %w", err)
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "inventory approved",
	}

	app.writeJSON(w, http.StatusAccepted, payload)

}

func (app *Config) AdminGetActiveSubscriptions(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.AdminGetActiveSubscriptionPayload
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

	// get the user rating and count
	data, err := app.Repo.AdminGetActiveSubscriptions(timeoutCtx, requestPayload)
	if err != nil {
		log.Fatal("error retrieving subscription: %w", err)
		app.errorJSON(w, err, nil)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved succesfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)

}

func (app *Config) AdminGetUsers(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.AdminGetUsersPayload
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Printf("%v", err)
		app.errorJSON(w, err, nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	users, err := app.Repo.GetAllUsers(ctx, requestPayload)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "users retrieved successfully",
		Data:       users,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) AdminGetDashboardCard(w http.ResponseWriter, r *http.Request) {

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.AdminGetDashboardCard(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) AdminGetAmountMadeByDate(w http.ResponseWriter, r *http.Request) {

	date := chi.URLParam(r, "date")

	if date == "" {
		app.errorJSON(w, errors.New("date parameter is missing"), nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.AdminGetAmountMadeByDate(ctx, date)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) AdminGetUsersJoinedByDate(w http.ResponseWriter, r *http.Request) {

	date := chi.URLParam(r, "date")

	if date == "" {
		app.errorJSON(w, errors.New("date parameter is missing"), nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.AdminGetUsersJoinedByDate(ctx, date)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) AdminGetInventoryCreatedByDate(w http.ResponseWriter, r *http.Request) {

	date := chi.URLParam(r, "date")

	if date == "" {
		app.errorJSON(w, errors.New("date parameter is missing"), nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.AdminGetInventoryCreatedByDate(ctx, date)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) GetUserRegistrationStats(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.RegistrationStatsRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Printf("%v", err)
		app.errorJSON(w, err, nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.GetUserRegistrationStats(ctx, requestPayload)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
func (app *Config) GetInventoryCreationStats(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.RegistrationStatsRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Printf("%v", err)
		app.errorJSON(w, err, nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.GetInventoryCreationStats(ctx, requestPayload)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) GetSubscriptionAmountStats(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload data.SubscriptionStatsRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Printf("%v", err)
		app.errorJSON(w, err, nil)
		return
	}

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	data, err := app.Repo.GetSubscriptionAmountStats(ctx, requestPayload)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "data retrieved successfully",
		Data:       data,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
