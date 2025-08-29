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
	mux.Post("/api/v1/my-booking", app.MyBookings)
	mux.Post("/api/v1/booking-requests", app.GetBookingRequest)
	mux.Post("/api/v1/my-inventories", app.MyInventories)
	mux.Post("/api/v1/my-subscription-history", app.MySubscriptionHistory)
	mux.Post("/api/v1/create-order", app.CreatePrurchaseOrder)
	mux.Post("/api/v1/my-purchase", app.MyPurchase)
	mux.Post("/api/v1/purchase-requests", app.GetPurchaseRequest)
	mux.Post("/api/v1/submit-chat", app.SubmitChat)
	mux.Post("/api/v1/chat-history", app.GetChatHistory)
	mux.Post("/api/v1/chat-list", app.GetChatList)
	mux.Post("/api/v1/unread-chat", app.GetUnreadChat)
	mux.Post("/api/v1/mark-chat-as-read", app.MarkChatAsRead)
	mux.Post("/api/v1/profile-image", app.UploadProfileImage)
	mux.Post("/api/v1/shop-banner", app.UploadBanner)
	mux.Post("/api/v1/save-inventory", app.SaveInventory)
	mux.Post("/api/v1/delete-saved-inventory", app.DeleteSaveInventory)
	mux.Post("/api/v1/delete-inventory", app.DeleteInventory)
	mux.Post("/api/v1/delete-chat", app.DeleteChat)

	mux.Post("/api/v1/report-user-rating", app.ReportUserRating)
	mux.Post("/api/v1/user-rating-helpful", app.UserRatingHelpful)

	mux.Post("/api/v1/report-inventory-rating", app.ReportInventoryRating)
	mux.Post("/api/v1/inventory-rating-helpful", app.InventoryRatingHelpful)

	mux.Post("/api/v1/user-saved-inventory", app.GetUserSavedInventory)

	mux.Get("/api/v1/user-detail", app.GetUserDetail)

	mux.Post("/api/v1/premium-partners", app.PremiumPartner)
	mux.Get("/api/v1/premium-extras", app.GetPremiumUsersExtras)

	mux.Get("/api/v1/business-details", app.GetBusinessDetail)

	mux.Get("/api/v1/inventory-rating-replies", app.GetInventoryRatingReplies)
	mux.Get("/api/v1/user-rating-replies", app.GetUserRatingReplies)
	// Add the Prometheus metrics endpoint to the router
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
