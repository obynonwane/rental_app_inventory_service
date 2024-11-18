package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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

	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return &inventory.CreateInventoryResponse{
			Message:    "Failed to initialize Cloudinary",
			StatusCode: 500,
			Error:      true,
		}, err
	}

	go func() {
		var urls []string
		var wg sync.WaitGroup

		for _, image := range req.Images {
			wg.Add(1)
			go func(img *inventory.ImageData) {
				defer wg.Done()

				uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				// Validate MIME type and map to Cloudinary's expected format
				switch img.ImageType {
				case "image/jpeg", "image/png", "image/gif": // Supported types
					// Generate unique filename (without extension)
					uniqueFilename := i.App.generateUniqueFilename()

					// Upload directly from byte stream to Cloudinary
					uploadResult, err := cld.Upload.Upload(uploadCtx, bytes.NewReader(img.ImageData), uploader.UploadParams{
						Folder:   "rentalsolution/inventories",
						PublicID: uniqueFilename, // Pass filename without extension
					})
					if err != nil {
						log.Printf("Error uploading to Cloudinary: %v", err)
						return
					}

					// Append the Cloudinary URL in a thread-safe manner
					i.App.mu.Lock()
					urls = append(urls, uploadResult.SecureURL)
					i.App.mu.Unlock()

				default:
					log.Printf("Unsupported image format: %s", img.ImageType)
					return
				}
			}(image)
		}

		wg.Wait()

		log.Println(urls, "the urls")

		// Save product details and images in the database (if applicable)
	}()

	// Immediately return success response to the user
	return &inventory.CreateInventoryResponse{
		Message:    "Inventory creation request received. Processing images in the background.",
		StatusCode: 202, // 202 Accepted since the processing is asynchronous
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
				Id:             category.ID,
				Name:           category.Name,
				Description:    category.Description,
				IconClass:      category.IconClass,
				CreatedAtHuman: formatTimestamp(timestamppb.New(category.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(category.UpdatedAt)),
			}

			allCategories = append(allCategories, singleCategory)
		}

		return &inventory.AllCategoryResponse{
			Categories: allCategories,
			StatusCode: http.StatusOK,
		}, nil

	case err := <-errorChannel:
		return nil, fmt.Errorf("failed to retrieve categories: %v", err)

	case <-timeoutCtx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching categories")
	}

}

func (i *InventoryServer) GetSubCategories(ctx context.Context, req *inventory.EmptyRequest) (*inventory.AllSubCategoryResponse, error) {

	subCategoriesChannel := make(chan []*data.Subcategory)
	errorChannel := make(chan error)

	// create a context with a timeout for the asynchronous task
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		subCategories, err := i.Models.GetAllSubCategory(timeoutCtx)
		if err != nil {
			errorChannel <- err // Send error to the error channel
			return
		}

		subCategoriesChannel <- subCategories
	}()

	select {
	case subCategories := <-subCategoriesChannel:

		// declare a map of type inventory category response of model type mismatch with the proto message type
		var allSubCategories []*inventory.SubCategoryResponse

		// loop and push response to above array
		for _, subCategory := range subCategories {

			singleSubCategory := &inventory.SubCategoryResponse{
				Id:             subCategory.ID,
				Name:           subCategory.Name,
				CategoryId:     subCategory.CategoryId,
				Description:    subCategory.Description,
				IconClass:      subCategory.IconClass,
				CreatedAtHuman: formatTimestamp(timestamppb.New(subCategory.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(subCategory.UpdatedAt)),
			}

			allSubCategories = append(allSubCategories, singleSubCategory)
		}

		return &inventory.AllSubCategoryResponse{
			Subcategories: allSubCategories,
			StatusCode:    http.StatusOK,
		}, nil

	case err := <-errorChannel:
		return nil, fmt.Errorf("failed to retrieve subcategories: %v", err)

	case <-timeoutCtx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching subcategories")
	}

}
func (i *InventoryServer) GetCategorySubcategories(ctx context.Context, req *inventory.ResourceId) (*inventory.AllSubCategoryResponse, error) {

	subCategoriesChannel := make(chan []*data.Subcategory)
	errorChannel := make(chan error)

	// create a context with a timeout for the asynchronous task
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		subCategories, err := i.Models.GetcategorySubcategories(timeoutCtx, req.Id)
		if err != nil {
			errorChannel <- err // Send error to the error channel
			return
		}

		subCategoriesChannel <- subCategories
	}()

	select {
	case subCategories := <-subCategoriesChannel:

		// declare a map of type inventory category response of model type mismatch with the proto message type
		var allSubCategories []*inventory.SubCategoryResponse

		// loop and push response to above array
		for _, subCategory := range subCategories {

			singleSubCategory := &inventory.SubCategoryResponse{
				Id:             subCategory.ID,
				Name:           subCategory.Name,
				CategoryId:     subCategory.CategoryId,
				Description:    subCategory.Description,
				IconClass:      subCategory.IconClass,
				CreatedAtHuman: formatTimestamp(timestamppb.New(subCategory.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(subCategory.UpdatedAt)),
			}

			allSubCategories = append(allSubCategories, singleSubCategory)
		}

		return &inventory.AllSubCategoryResponse{
			Subcategories: allSubCategories,
			StatusCode:    http.StatusOK,
		}, nil

	case err := <-errorChannel:
		return nil, fmt.Errorf("failed to retrieve subcategories: %v", err)

	case <-timeoutCtx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching subcategories")
	}

}

func (i *InventoryServer) GetCategory(ctx context.Context, req *inventory.ResourceId) (*inventory.CategoryResponse, error) {

	// intantiate response and error channels
	categoryChannel := make(chan *data.Category)
	erroChannel := make(chan error)

	//create a context with timeout for asynchronous operation
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// start go routin for asynchronous execution
	go func() {
		data, err := i.Models.GetcategoryByID(ctx, req.Id)
		if err != nil {
			erroChannel <- err
			return
		}

		categoryChannel <- data
	}()

	select {
	case category := <-categoryChannel:
		return &inventory.CategoryResponse{
			Id:             category.ID,
			Name:           category.Name,
			Description:    category.Description,
			IconClass:      category.IconClass,
			CreatedAtHuman: formatTimestamp(timestamppb.New(category.CreatedAt)),
			UpdatedAtHuman: formatTimestamp(timestamppb.New(category.UpdatedAt)),
			StatusCode:     http.StatusOK,
		}, nil

	case err := <-erroChannel:
		return nil, fmt.Errorf("failed to retrieve category: %v", err)

	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("request timed out while fetching subcategories")
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
				Id:             user.ID,
				Email:          user.Email,
				FirstName:      user.FirstName,
				LastName:       user.LastName,
				Verified:       user.Verified,
				CreatedAtHuman: formatTimestamp(timestamppb.New(user.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(user.UpdatedAt)),
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
