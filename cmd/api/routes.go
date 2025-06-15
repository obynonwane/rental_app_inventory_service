package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/* returns http.Handler*/
func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	//specify who is allowed to connect
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Get("/api/v1/getusers", app.GetUsers)
	mux.Post("/api/v1/create-booking", app.CreateBooking)
	mux.Post("/api/v1/create-order", app.CreatePrurchaseOrder)
	mux.Post("/api/v1/submit-chat", app.SubmitChat)
	// Add the Prometheus metrics endpoint to the router
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
