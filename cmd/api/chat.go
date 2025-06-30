package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	"github.com/obynonwane/inventory-service/data"
)

// func (app *Config) SubmitChat(w http.ResponseWriter, r *http.Request) {

// 	//extract the request body
// 	var requestPayload data.Message
// 	err := app.readJSON(w, r, &requestPayload)
// 	if err != nil {
// 		app.errorJSON(w, err, nil)
// 		return
// 	}

// 	// get the inventory

// 	// Create a context with a timeout for the asynchronous task
// 	ctx := r.Context()
// 	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
// 	defer cancel()

// 	chat, err := app.Repo.SubmitChat(timeoutCtx, &data.Message{
// 		Content:  requestPayload.Content,
// 		Sender:   requestPayload.Sender,
// 		Receiver: requestPayload.Receiver,
// 		SentAt:   requestPayload.SentAt,
// 	})
// 	if err != nil {
// 		app.errorJSON(w, err, nil, http.StatusInternalServerError)
// 		return
// 	}

// 	// send sms & email notification to both owner and buyer

// 	payload := jsonResponse{
// 		Error:      false,
// 		StatusCode: http.StatusAccepted,
// 		Message:    "chat created successfully",
// 		Data:       chat,
// 	}

// 	app.writeJSON(w, http.StatusAccepted, payload)
// }

func (app *Config) DetectContentType(content string) (string, string) {
	log.Println("Detecting content type")

	if strings.HasPrefix(content, "data:") {
		header := strings.Split(strings.Split(content, ";")[0], ":")[1]

		switch {
		case strings.HasPrefix(header, "image/"):
			return "image", header
		case strings.HasPrefix(header, "application/"):
			return "file", header
		case strings.HasPrefix(header, "video/"):
			return "video", header
		}
	}

	return "text", "text/plain"
}

func (app *Config) SubmitChat(w http.ResponseWriter, r *http.Request) {

	var requestPayload data.Message
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		log.Println("Error reading JSON:", err)
		app.errorJSON(w, err, nil)
		return
	}

	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msgType, mimeType := app.DetectContentType(requestPayload.Content)

	switch msgType {
	case "image", "file", "video":
		log.Printf("Processing as %s", msgType)

		cld, err := cloudinary.NewFromParams(
			os.Getenv("CLOUDINARY_CLOUD_NAME"),
			os.Getenv("CLOUDINARY_API_KEY"),
			os.Getenv("CLOUDINARY_API_SECRET"),
		)
		if err != nil {
			log.Println("Error initializing Cloudinary:", err)
			app.errorJSON(w, err, nil)
			return
		}

		uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		uniqueFilename := app.generateUniqueFilename()

		// Strip base64 prefix
		parts := strings.SplitN(requestPayload.Content, ",", 2)
		if len(parts) != 2 {
			log.Println("Invalid base64 format")
			app.errorJSON(w, errors.New("invalid base64 format"), nil)
			return
		}
		base64Data := parts[1]

		decoded, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			log.Println("Base64 decode error:", err)
			app.errorJSON(w, err, nil)
			return
		}

		// Choose Cloudinary resource type
		resourceType := map[string]string{
			"image": "image",
			"file":  "raw",
			"video": "video",
		}[msgType]

		uploadResult, err := cld.Upload.Upload(uploadCtx, bytes.NewReader(decoded), uploader.UploadParams{
			Folder:       "rentalsolution/chats",
			PublicID:     uniqueFilename,
			ResourceType: resourceType,
		})
		if err != nil {
			log.Printf("%s upload failed: %v", msgType, err)
			app.errorJSON(w, err, nil)
			return
		}

		log.Printf("Uploaded to Cloudinary: %s", uploadResult.SecureURL)
		requestPayload.Content = uploadResult.SecureURL

	case "text":
		log.Println("Processing as plain text")
		// Content stays as-is
	default:
		log.Println("Unsupported message type:", msgType)
		app.errorJSON(w, errors.New("unsupported content type"), nil)
		return
	}

	// Save chat to database
	chat, err := app.Repo.SubmitChat(timeoutCtx, &data.Message{
		Content:     requestPayload.Content,
		Sender:      requestPayload.Sender,
		ReplyTo:     requestPayload.ReplyTo,
		Receiver:    requestPayload.Receiver,
		SentAt:      requestPayload.SentAt,
		Type:        msgType,
		ContentType: mimeType,
	})
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	app.writeJSON(w, http.StatusAccepted, jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "chat created successfully",
		Data:       chat,
	})
}

// Define the payload structure

type ChatHistoryRequest struct {
	UserA string `json:"userA"`
	UserB string `json:"userB"`
}

func (app *Config) GetChatHistory(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload ChatHistoryRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	chat, err := app.Repo.GetChatHistory(timeoutCtx, requestPayload.UserA, requestPayload.UserB)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	if chat == nil {
		chat = []data.Chat{}
	}

	// send sms & email notification to both owner and buyer
	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "chat history retrieved successfully",
		Data:       chat,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

type ChatListRequest struct {
	UserID string `json:"user_id"`
}

func (app *Config) GetChatList(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload ChatListRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	chat, err := app.Repo.GetChatList(timeoutCtx, requestPayload.UserID)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	if chat == nil {
		chat = []data.ChatSummary{}
	}

	// send sms & email notification to both owner and buyer
	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "chat list retrieved successfully",
		Data:       chat,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

type UnreadChatRequest struct {
	UserID string `json:"user_id"`
}

func (app *Config) GetUnreadChat(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload UnreadChatRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	chat, err := app.Repo.GetUnreadChat(timeoutCtx, requestPayload.UserID)
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
		Message:    "unread chat retrieved successfully",
		Data:       chat,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

type MarkChatRequest struct {
	UserID   string `json:"user_id"`
	SenderID string `json:"sender_id"`
}

func (app *Config) MarkChatAsRead(w http.ResponseWriter, r *http.Request) {

	//extract the request body
	var requestPayload MarkChatRequest
	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, nil)
		return
	}

	// Create a context with a timeout for the asynchronous task
	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	err = app.Repo.MarkChatAsRead(timeoutCtx, requestPayload.UserID, requestPayload.SenderID)
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
		Message:    "chat marked successfully",
		Data:       nil,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
