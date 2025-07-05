package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
)

func (app *Config) UploadProfileImage(w http.ResponseWriter, r *http.Request) {
	// Parse the incoming multipart form
	err := r.ParseMultipartForm(20 << 20) // 20 MB
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract the file
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Extract user ID
	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	// Read image into a buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	log.Println(userID)
	log.Println(file)

	// Increase the timeout duration for Cloudinary initialization and image uploads
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		if err != nil {
			app.errorJSON(w, err, nil, http.StatusInternalServerError)
			return
		}
	}

	// Generate unique filename (without extension)
	uniqueFilename := app.generateUniqueFilename()

	uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Increased timeout for image upload
	defer cancel()

	// Upload directly from byte stream to Cloudinary
	uploadResult, err := cld.Upload.Upload(uploadCtx, bytes.NewReader(buf.Bytes()), uploader.UploadParams{
		Folder:   "rentalsolution/profile_images",
		PublicID: uniqueFilename, // Pass filename without extension
	})
	if err != nil {
		log.Printf("Error uploading to Cloudinary: %v", err)
		return
	}

	// log.Println(uploadResult.SecureURL)
	imageUrl := uploadResult.SecureURL

	err = app.Repo.UploadProfileImage(timeoutCtx, imageUrl, userID)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}
	app.writeJSON(w, http.StatusAccepted, jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "profile image uploaded successfully",
	})
}

func (app *Config) UploadBanner(w http.ResponseWriter, r *http.Request) {
	// Parse the incoming multipart form
	err := r.ParseMultipartForm(20 << 20) // 20 MB
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract the file
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ctx := r.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Extract user ID
	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	// Read image into a buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	log.Println(userID)
	log.Println(file)

	// Increase the timeout duration for Cloudinary initialization and image uploads
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		if err != nil {
			app.errorJSON(w, err, nil, http.StatusInternalServerError)
			return
		}
	}

	// Generate unique filename (without extension)
	uniqueFilename := app.generateUniqueFilename()

	uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Increased timeout for image upload
	defer cancel()

	// Upload directly from byte stream to Cloudinary
	uploadResult, err := cld.Upload.Upload(uploadCtx, bytes.NewReader(buf.Bytes()), uploader.UploadParams{
		Folder:   "rentalsolution/business_banners",
		PublicID: uniqueFilename, // Pass filename without extension
	})
	if err != nil {
		log.Printf("Error uploading to Cloudinary: %v", err)
		return
	}

	log.Println(uploadResult.SecureURL)
	imageUrl := uploadResult.SecureURL

	err = app.Repo.UploadProfileImage(timeoutCtx, imageUrl, userID)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}
	app.writeJSON(w, http.StatusAccepted, jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "profile image uploaded successfully",
	})
}
