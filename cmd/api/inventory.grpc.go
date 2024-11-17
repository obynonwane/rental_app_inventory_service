package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	"github.com/obynonwane/inventory-service/data"
	"github.com/obynonwane/rental-service-proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// create the Inventory erver
type InventoryServer struct {
	inventory.UnimplementedInventoryServiceServer
	Models data.Repository
	App    *Config
}

func (i *InventoryServer) CreateInventory(ctx context.Context, req *inventory.CreateInventoryRequest) (*inventory.CreateInventoryResponse, error) {

	imagesDir := "inventory-service/uploads"
	os.MkdirAll(imagesDir, os.ModePerm)

	cld, err := cloudinary.NewFromParams(os.Getenv("CLOUDINARY_CLOUD_NAME"), os.Getenv("CLOUDINARY_API_KEY"), os.Getenv("CLOUDINARY_API_SECRET"))
	if err != nil {
		return &inventory.CreateInventoryResponse{
			Message:    "Failed to initialize Cloudinary",
			StatusCode: 500,
			Error:      true,
		}, err
	}

	// Launch a goroutine to process images asynchronously
	go func() {
		var wg sync.WaitGroup
		for _, image := range req.Images {
			wg.Add(1)
			go func(img *inventory.ImageData) {
				defer wg.Done()

				// Create a new context for the Cloudinary upload to avoid cancellation issues
				uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Set a sufficient timeout for the upload
				defer cancel()

				// Determine file extension based on MIME type
				var ext string
				switch img.ImageType {
				case "image/jpeg":
					ext = "jpg"
				case "image/png":
					ext = "png"
				case "image/gif":
					ext = "gif"
				default:
					log.Printf("Unsupported image format: %s", img.ImageType)
					return
				}

				uniqueFilename := i.App.generateUniqueFilename(ext)
				filePath := filepath.Join(imagesDir, uniqueFilename)

				// Save the file locally
				err := os.WriteFile(filePath, img.ImageData, 0644)
				if err != nil {
					log.Printf("Error saving image: %v", err)
					return
				}

				// Upload to Cloudinary with the new context
				uploadResult, err := cld.Upload.Upload(uploadCtx, filePath, uploader.UploadParams{
					Folder:   "rentalsolution/inventories", // Optional folder name in Cloudinary
					PublicID: uniqueFilename,
				})
				if err != nil {
					log.Printf("Error uploading to Cloudinary: %v", err)
					return
				}

				// Store the Cloudinary URL in the database
				log.Println("Cloudinary upload successful, URL: ", uploadResult.SecureURL)

				// Clean up local file after upload
				os.Remove(filePath)
			}(image)
		}

		// Wait for all uploads to complete
		wg.Wait()
		log.Println("All images have been uploaded to Cloudinary and URLs stored in the database.")
	}()

	// Return success response
	return &inventory.CreateInventoryResponse{
		Message:    "Inventory created successfully",
		StatusCode: 200,
		Error:      false,
	}, nil
}

// func (i *InventoryServer) GetUsers(ctx context.Context, req *inventory.EmptyRequest) (*inventory.UserListResponse, error) {

// 	users, err := i.Models.GetAll(ctx)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, err
// 		}
// 	}

// 	// Convert []*data.User to []*inventory.User
// 	var inventoryUsers []*inventory.User
// 	for _, user := range users {
// 		invUser := &inventory.User{
// 			Id:        user.ID,
// 			Email:     user.Email,
// 			FirstName: user.FirstName,
// 			LastName:  user.LastName,
// 			Verified:  user.Verified,
// 			CreatedAt: timestamppb.New(user.CreatedAt), // assuming google.protobuf.Timestamp is used
// 			UpdatedAt: timestamppb.New(user.UpdatedAt),
// 		}
// 		inventoryUsers = append(inventoryUsers, invUser)
// 	}

// 	response := &inventory.UserListResponse{
// 		Users: inventoryUsers,
// 	}
// 	return response, nil

// }

func (i *InventoryServer) GetCategories(ctx context.Context, req *inventory.EmptyRequest) (*inventory.AllCategoryResponse, error) {

	categoriesChannel := make(chan []*data.Category)
	errorChannel := make(chan error)

	// create a context with a timeout for the asynchronous task
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		categories, err := i.Models.GetAllCategory(timeoutCtx)
		if err != nil {
			errorChannel <- err // Send error to the error channel
			return
		}

		categoriesChannel <- categories
	}()

	select {
	case categories := <-categoriesChannel:

		// declare a map of type inventory category response of model type mismatch with the proto message type
		var allCategories []*inventory.CategoryResponse

		// loop and push response to above array
		for _, category := range categories {
			singleCategory := &inventory.CategoryResponse{
				Id:          category.ID,
				Name:        category.Name,
				Description: category.Description,
				IconClass:   category.IconClass,
				CreatedAt:   timestamppb.New(category.CreatedAt),
				UpdatedAt:   timestamppb.New(category.UpdatedAt),
			}

			allCategories = append(allCategories, singleCategory)
		}

		return &inventory.AllCategoryResponse{
			Categories: allCategories,
		}, nil

	case err := <-errorChannel:
		return nil, fmt.Errorf("failed to retrieve categories: %v", err)

	case <-timeoutCtx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching categories")
	}

}

func (i *InventoryServer) GetUsers(ctx context.Context, req *inventory.EmptyRequest) (*inventory.UserListResponse, error) {
	// Create a channel to signal completion of the async task
	userChannel := make(chan []*data.User)
	errorChannel := make(chan error)

	// Create a context with a timeout for the asynchronous task
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	// Perform the database query asynchronously in a goroutine
	go func() {
		users, err := i.Models.GetAll(timeoutCtx) // Pass the timeout context to the DB call
		if err != nil {
			errorChannel <- err // Send error to the error channel
			return
		}
		userChannel <- users // Send the users to the user channel
	}()

	// Wait for either the users, an error, or a timeout to occur
	select {
	case users := <-userChannel:
		// Process the users and prepare the response
		var inventoryUsers []*inventory.User
		for _, user := range users {
			invUser := &inventory.User{
				Id:        user.ID,
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Verified:  user.Verified,
				CreatedAt: timestamppb.New(user.CreatedAt),
				UpdatedAt: timestamppb.New(user.UpdatedAt),
			}
			inventoryUsers = append(inventoryUsers, invUser)
		}

		response := &inventory.UserListResponse{
			Users: inventoryUsers,
		}
		return response, nil

	case err := <-errorChannel:
		// If there was an error fetching users, return it
		return nil, fmt.Errorf("failed to retrieve users: %v", err)

	case <-timeoutCtx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching users")
	}
}

// start listening to tcp connection
func (app *Config) grpcListen() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gRpcPort))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	s := grpc.NewServer()
	inventory.RegisterInventoryServiceServer(s, &InventoryServer{Models: app.Repo, App: app})

	log.Printf("gRPC Server started on port %s", gRpcPort)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}
}
