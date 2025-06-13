package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
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
	"github.com/obynonwane/inventory-service/utility"
	"github.com/obynonwane/rental-service-proto/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Define a custom type for the context key

// create the Inventory erver
type InventoryServer struct {
	inventory.UnimplementedInventoryServiceServer
	Models data.Repository
	App    *Config
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

func (app *Config) GetUsers(w http.ResponseWriter, r *http.Request) {

	// Extract the context from the incoming HTTP request
	ctx := r.Context()

	users, err := app.Repo.GetAll(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	log.Println(users)

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "users retrieved successfully",
		Data:       users,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

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
	EndTime   string `json:"end_time"`                      // e.g., "18:00:00", optional for daily+ rentals

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
		app.errorJSON(w, errors.New(fmt.Sprintf("the stipulated quantity is not available, only: %v", inv.Quantity, " is available")), nil, http.StatusBadRequest)
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

	endTime, err := time.Parse(timeLayout, requestPayload.EndTime)
	if err != nil {
		app.errorJSON(w, fmt.Errorf("invalid end time format, use HH:MM (24-hour format)"), nil, http.StatusBadRequest)
		return
	}

	// making sure the end date and start date is not in the past
	err = utility.ValidateBookingDates(startDate, endDate)
	if err != nil {
		app.errorJSON(w, err, nil, http.StatusBadRequest)
		return
	}

	// calculate the total amount (quantity * offer_price_per_unit)
	totalPrice := requestPayload.Quantity * requestPayload.OfferPricePerUnit

	_, err = app.Repo.CreateBooking(timeoutCtx, &data.CreateBookingPayload{
		InventoryId:       inv.ID,
		RentalType:        requestPayload.RentalType,
		RentalDuration:    int32(requestPayload.RentalDuration),
		SecurityDeposit:   requestPayload.SecurityDeposit,
		OfferPricePerUnit: requestPayload.OfferPricePerUnit,
		Quantity:          int32(requestPayload.Quantity),
		TotalAmount:       totalPrice,
		StartDate:         startDate,
		EndDate:           endDate,
		EndTime:           endTime,
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
		Data:       inv,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (i *InventoryServer) CreateInventory(ctx context.Context, req *inventory.CreateInventoryRequest) (*inventory.CreateInventoryResponse, error) {

	var wg sync.WaitGroup

	catErrCh := make(chan error, 1)            // Buffered to avoid blocking
	categoryCh := make(chan *data.Category, 1) // Buffered to avoid blocking

	subCatErrCh := make(chan error, 1)          // Buffered to avoid blocking
	subCatCh := make(chan *data.Subcategory, 1) // Buffered to avoid blocking

	stateErrCh := make(chan error, 1)    // Buffered to avoid blocking
	stateCh := make(chan *data.State, 1) // Buffered to avoid blocking

	countryErrCh := make(chan error, 1)      // Buffered to avoid blocking
	countryCh := make(chan *data.Country, 1) // Buffered to avoid blocking

	lgaErrCh := make(chan error, 1)  // Buffered to avoid blocking
	lgaCh := make(chan *data.Lga, 1) // Buffered to avoid blocking

	// Validate category
	wg.Add(1)
	go func() {
		defer wg.Done()

		param := &data.GetCategoryByIDPayload{
			CategoryID: req.CategoryId,
		}

		category, catErr := i.Models.GetCategoryByID(ctx, param)
		catErrCh <- catErr     // Write the error or nil
		categoryCh <- category // Write the subcategory or nil
	}()

	// Validate subcategory
	wg.Add(1)
	go func() {
		defer wg.Done()
		subcategory, subCatErr := i.Models.GetSubcategoryByID(ctx, req.SubCategoryId)
		subCatErrCh <- subCatErr // Write the error or nil
		subCatCh <- subcategory  // Write the subcategory or nil
	}()

	// Validate state
	wg.Add(1)
	go func() {
		defer wg.Done()
		state, stateErr := i.Models.GetStateByID(ctx, req.StateId)
		stateErrCh <- stateErr // Write the error or nil
		stateCh <- state       // Write the state or nil
	}()

	// Validate country
	wg.Add(1)
	go func() {
		defer wg.Done()
		country, countryErr := i.Models.GetCountryByID(ctx, req.CountryId)
		countryErrCh <- countryErr // Write the error or nil
		countryCh <- country       // Write the country or nil
	}()

	// Validate Lga
	wg.Add(1)
	go func() {
		defer wg.Done()
		lga, lgaErr := i.Models.GetLgaByID(ctx, req.LgaId)
		lgaErrCh <- lgaErr // Write the error or nil
		lgaCh <- lga       // Write the country or nil
	}()

	// Wait for both goroutines to finish
	wg.Wait()

	// Close channels after all goroutines finish writing
	close(catErrCh)
	close(subCatErrCh)
	close(stateErrCh)
	close(countryErrCh)
	close(lgaErrCh)
	close(subCatCh)
	close(stateCh)
	close(countryCh)
	close(lgaCh)

	// Read category validation error
	catErr := <-catErrCh
	if catErr != nil {
		return nil, fmt.Errorf("error validating category: %v", catErr)
	}

	// Read subcategory validation error
	subCatErr := <-subCatErrCh
	if subCatErr != nil {
		return nil, fmt.Errorf("error validating subcategory: %v", subCatErr)
	}

	// Read state validation error
	stateErr := <-stateErrCh
	if stateErr != nil {
		return nil, fmt.Errorf("error validating state: %v", stateErr)
	}

	// Read country validation error
	countryErr := <-countryErrCh
	if countryErr != nil {
		return nil, fmt.Errorf("error validating country: %v", stateErr)
	}

	// Read lga validation error
	lgaErr := <-lgaErrCh
	if lgaErr != nil {
		return nil, fmt.Errorf("error validating lga: %v", lgaErr)
	}

	category := <-categoryCh
	country := <-countryCh
	// Read and validate subcategory
	subcategory := <-subCatCh
	if subcategory == nil {
		return nil, fmt.Errorf("subcategory not found")
	}

	if subcategory.CategoryId != req.CategoryId {
		return nil, fmt.Errorf("subcategory does not belong to category")
	}

	// Read and validate state & country
	state := <-stateCh
	if state == nil {
		return nil, fmt.Errorf("state not found")
	}
	if state.CountryID != req.CountryId {
		return nil, fmt.Errorf("state does not belong to country")
	}

	// Read and validate state & lga
	lga := <-lgaCh
	if lga == nil {
		return nil, fmt.Errorf("lga not found")
	}
	if lga.StateID != req.StateId {
		return nil, fmt.Errorf("lga does not belong to state")
	}

	// Increase the timeout duration for Cloudinary initialization and image uploads
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
		var primageImageURL string
		var wg sync.WaitGroup

		//======================Upload multiple Image  for inventories to cloudinary =====================================================================

		for _, image := range req.Images {
			wg.Add(1)
			go func(img *inventory.ImageData) {
				defer wg.Done()

				uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Increased timeout for image upload
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

					var mu sync.Mutex
					// Append the Cloudinary URL in a thread-safe manner
					mu.Lock()
					urls = append(urls, uploadResult.SecureURL)
					mu.Unlock()
				default:
					log.Printf("Unsupported image format: %s", img.ImageType)
					return
				}
			}(image)
		}

		//======================End Upload multiple Image  for inventories to cloudinary =====================================================================

		//======================Upload Single Image for - Primary Image to cloudinary =====================================================================
		wg.Add(1)
		go func(img *inventory.ImageData) {
			defer wg.Done()

			uploadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Increased timeout for image upload
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

				var mu sync.Mutex
				// Append the Cloudinary URL in a thread-safe manner
				mu.Lock()
				primageImageURL = uploadResult.SecureURL
				mu.Unlock()

			default:
				log.Printf("Unsupported image format: %s", img.ImageType)
				return
			}
		}(req.PrimaryImage)
		//====================== END Upload Single Image for - Primary Image to cloudinary =====================================================================

		wg.Wait()

		dbCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute) // Increased timeout for DB transaction
		defer cancel()

		// Prepare for transaction
		tx, err := i.Models.BeginTransaction(dbCtx)
		if err != nil {
			log.Println(fmt.Errorf("failed to begin transaction: %v", err))
			return
		}
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				panic(p)
			} else if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}()

		slug, ulid := utility.GenerateSlug(req.Name)
		description := utility.TextToLower(req.Description)

		// Save product details and images in the database (if applicable)
		err = i.Models.CreateInventory(&data.CreateInventoryParams{
			Tx:              tx,
			Ctx:             dbCtx,
			Name:            req.Name,
			Description:     description,
			UserID:          req.UserId,
			CategoryID:      req.CategoryId,
			SubcategoryID:   req.SubCategoryId,
			CountryID:       req.CountryId,
			StateID:         req.StateId,
			LgaID:           req.LgaId,
			Slug:            slug,
			ULID:            ulid,
			StateSlug:       state.StateSlug,
			CountrySlug:     country.Code,
			LgaSlug:         lga.LgaSlug,
			CategorySlug:    category.CategorySlug,
			SubcategorySlug: subcategory.SubCategorySlug,
			OfferPrice:      req.OfferPrice,
			MinimumPrice:    req.MinimumPrice,
			URLs:            urls,

			ProductPurpose:  req.ProductPurpose,
			Quantity:        req.Quantity,
			IsAvailable:     req.IsAvailable,
			RentalDuration:  req.RentalDuration,
			SecurityDeposit: req.SecurityDeposit,
			Tags:            req.Tags,
			Metadata:        req.Metadata,
			Negotiable:      req.Negotiable,
			PrimaryImage:    primageImageURL,
		})

		if err != nil {
			log.Println(fmt.Errorf("error creating inventory for user %s", req.UserId))
			return
		}
	}()

	// Immediately return success response to the user
	return &inventory.CreateInventoryResponse{
		Message:    "Inventory creation request received. Processing images in the background.",
		StatusCode: 202, // 202 Accepted since the processing is asynchronous
		Error:      false,
	}, nil
}

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

		// loop and push response to above allCategories slice
		for _, category := range categories {

			// declare slice of subcategory
			var subcategories []*inventory.SubCategoryResponse

			// loop through the subcategory as return
			for _, sub := range category.Subcategories {

				// append subcategory to subcategories slice
				subcategories = append(subcategories, &inventory.SubCategoryResponse{
					Id:              sub.ID,
					Name:            sub.Name,
					Description:     sub.Description,
					IconClass:       sub.IconClass,
					SubcategorySlug: sub.SubCategorySlug,
					CreatedAtHuman:  formatTimestamp(timestamppb.New(sub.CreatedAt)),
					UpdatedAtHuman:  formatTimestamp(timestamppb.New(sub.UpdatedAt)),
				})
			}

			singleCategory := &inventory.CategoryResponse{
				Id:             category.ID,
				Name:           category.Name,
				Description:    category.Description,
				IconClass:      category.IconClass,
				CategorySlug:   category.CategorySlug,
				CreatedAtHuman: formatTimestamp(timestamppb.New(category.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(category.UpdatedAt)),
				Subcategories:  subcategories,
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
				Id:              subCategory.ID,
				Name:            subCategory.Name,
				CategoryId:      subCategory.CategoryId,
				Description:     subCategory.Description,
				IconClass:       subCategory.IconClass,
				SubcategorySlug: subCategory.SubCategorySlug,
				CreatedAtHuman:  formatTimestamp(timestamppb.New(subCategory.CreatedAt)),
				UpdatedAtHuman:  formatTimestamp(timestamppb.New(subCategory.UpdatedAt)),
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
				Id:              subCategory.ID,
				Name:            subCategory.Name,
				CategoryId:      subCategory.CategoryId,
				Description:     subCategory.Description,
				IconClass:       subCategory.IconClass,
				SubcategorySlug: subCategory.SubCategorySlug,
				CreatedAtHuman:  formatTimestamp(timestamppb.New(subCategory.CreatedAt)),
				UpdatedAtHuman:  formatTimestamp(timestamppb.New(subCategory.UpdatedAt)),
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

func (i *InventoryServer) GetCategory(ctx context.Context, req *inventory.GetCategoryByIDPayload) (*inventory.CategoryResponse, error) {

	// intantiate response and error channels
	categoryChannel := make(chan *data.Category)
	erroChannel := make(chan error)

	//create a context with timeout for asynchronous operation
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	param := &data.GetCategoryByIDPayload{
		CategoryID:   req.CategoryId,
		CategorySlug: req.CategorySlug,
	}

	// start go routin for asynchronous execution
	go func() {
		data, err := i.Models.GetCategoryByID(ctx, param)
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
			CategorySlug:   category.CategorySlug,
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

func (i *InventoryServer) RateInventory(ctx context.Context, req *inventory.InventoryRatingRequest) (*inventory.InventoryRatingResponse, error) {
	// start a work group to retrive the user who owns the inventory
	var wg sync.WaitGroup
	inventoryOwnerCh := make(chan *data.Inventory, 1)
	inventoryErr := make(chan error, 1)

	// Create a context with a timeout for the asynchronous task
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		inv, err := i.Models.GetInventoryByID(timeoutCtx, req.InventoryId)

		if err != nil {
			inventoryErr <- err
		}
		inventoryOwnerCh <- inv

	}()

	wg.Wait()
	close(inventoryOwnerCh)
	close(inventoryErr)

	retrievedInventory := <-inventoryOwnerCh
	theErr := <-inventoryErr

	if theErr != nil {
		return nil, fmt.Errorf("failed to retrieve inventory: %v", theErr)
	}

	createdRatingCh := make(chan *data.InventoryRating)
	ratingCreateErrCh := make(chan error)
	// start a gorouting for creating the inventory
	go func() {
		// make call to db to create inventory
		createdInv, err := i.Models.CreateInventoryRating(ctx, req.InventoryId, req.RaterId, retrievedInventory.UserId, req.Comment, req.Rating)
		if err != nil {
			ratingCreateErrCh <- err
		}

		createdRatingCh <- createdInv
	}()

	select {
	case data := <-createdRatingCh:
		return &inventory.InventoryRatingResponse{
			Id:             data.ID,
			InventoryId:    data.InventoryId,
			RaterId:        data.RaterId,
			Rating:         data.Rating,
			Comment:        data.Comment,
			CreatedAtHuman: formatTimestamp(timestamppb.New(data.CreatedAt)),
			UpdatedAtHuman: formatTimestamp(timestamppb.New(data.UpdatedAt)),
		}, nil

	case err := <-ratingCreateErrCh:
		return nil, fmt.Errorf("failed to create inventory rating: %v", err)

	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("request timed out while fetching subcategories")
	}

}

func (i *InventoryServer) RateUser(ctx context.Context, req *inventory.UserRatingRequest) (*inventory.UserRatingResponse, error) {

	// channel to check if user been rated exist
	userExist := make(chan *data.User, 1)
	userErr := make(chan error, 1)

	// channel to create the rating
	createdRatingCh := make(chan *data.UserRating, 1)
	ratingCreateErr := make(chan error, 1)

	// Create a context with a timeout for the asynchronous task
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Example timeout duration
	defer cancel()

	go func() {
		user, err := i.Models.GetUserByID(ctx, req.UserId)
		if err != nil {
			userErr <- err
		}
		userExist <- user
	}()

	select {
	case data := <-userExist:
		go func() {
			createdUserRating, err := i.Models.CreateUserRating(timeoutCtx, data.ID, req.Rating, req.Comment, req.RaterId)
			if err != nil {
				ratingCreateErr <- err
			}
			createdRatingCh <- createdUserRating
		}()

		select {
		case rating := <-createdRatingCh:
			return &inventory.UserRatingResponse{
				Id:             rating.ID,
				UserId:         rating.UserId,
				RaterId:        rating.RaterId,
				Rating:         rating.Rating,
				Comment:        rating.Comment,
				CreatedAtHuman: formatTimestamp(timestamppb.New(rating.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(rating.UpdatedAt)),
			}, nil

		case err := <-userErr:
			return nil, fmt.Errorf("failed to create rating for user: %v", err)

		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("request timed out while creating rating")
		}

	case err := <-userErr:
		// If there was an error fetching users, return it
		log.Println(fmt.Errorf("failed to retrieve user who is been rated: %v", err))
		return nil, fmt.Errorf("failed to retrieve user who is been rated")

	case <-ctx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching user who is been rated")
	}
}

func (i *InventoryServer) GetInventoryByID(ctx context.Context, req *inventory.SingleInventoryRequestDetail) (*inventory.InventoryResponseDetail, error) {

	// 1. channel to hold inventory & error channel
	inventoryExistCh := make(chan *data.Inventory, 1)
	errInventoryExistCh := make(chan error, 1)

	// 2. create a timeout to make sure this ha
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 3. go routine to retrieve inventory whose id is supplied
	go func() {

		inv, err := i.Models.GetInventoryByIDOrSlug(timeoutCtx, req.SlugUlid, req.InventoryId)

		if err != nil {
			errInventoryExistCh <- err
		}
		inventoryExistCh <- inv

	}()

	select {

	case di := <-inventoryExistCh:

		user, err := i.Models.GetUserByID(ctx, di.UserId)
		if err != nil {
			log.Fatal("error getting user who owns inventory", err)
		}

		// get the country
		country, err := i.Models.GetCountryByID(ctx, di.CountryId)
		if err != nil {
			log.Fatal("error retrieving country: %w", err)
		}

		// get the state
		state, err := i.Models.GetStateByID(ctx, di.StateId)
		if err != nil {
			log.Fatal("error retrieving state: %w", err)
		}

		// get the lga
		lga, err := i.Models.GetLgaByID(ctx, di.LgaId)
		if err != nil {
			log.Fatal("error retrieving lga: %w", err)
		}

		category, err := i.Models.GetCategoryByID(ctx, &data.GetCategoryByIDPayload{CategoryID: di.CategoryId})
		if err != nil {
			log.Fatal("error retrieving category: %w", err)
		}

		subcategory, err := i.Models.GetSubcategoryByID(ctx, di.SubcategoryId)
		if err != nil {
			log.Fatal("error retrieving subcategory: %w", err)
		}

		log.Println("MINIMUM PRICE", di.MinimumPrice)
		return &inventory.InventoryResponseDetail{
			Inventory: &inventory.Inventory{

				Id:             di.ID,
				Name:           di.Name,
				Description:    di.Description,
				UserId:         di.UserId,
				CategoryId:     di.CategoryId,
				SubcategoryId:  di.SubcategoryId,
				Promoted:       di.Promoted,
				Deactivated:    di.Deactivated,
				CreatedAt:      timestamppb.New(di.CreatedAt),
				UpdatedAt:      timestamppb.New(di.UpdatedAt),
				CreatedAtHuman: formatTimestamp(timestamppb.New(di.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(di.UpdatedAt)),
				Slug:           di.Slug,
				Ulid:           di.Ulid,
				OfferPrice:     di.OfferPrice,

				ProductPurpose:  di.ProductPurpose,
				Quantity:        di.Quantity,
				IsAvailable:     di.IsAvailable,
				RentalDuration:  di.RentalDuration,
				SecurityDeposit: di.SecurityDeposit,
				Metadata:        di.Metadata,
				Negotiable:      di.Negotiable,
				PrimaryImage:    di.PrimaryImage,
				MinimumPrice:    di.MinimumPrice,

				CountryId: di.CountryId,
				StateId:   di.StateId,
				LgaId:     di.LgaId,

				StateSlug:       di.StateSlug,
				CountrySlug:     di.CountrySlug,
				LgaSlug:         di.LgaSlug,
				CategorySlug:    di.CategorySlug,
				SubcategorySlug: di.SubcategorySlug,
			},

			User: &inventory.User{
				Id:             user.ID,
				FirstName:      user.FirstName,
				LastName:       user.LastName,
				Phone:          user.Phone,
				Email:          user.Email,
				Verified:       user.Verified,
				CreatedAtHuman: formatTimestamp(timestamppb.New(user.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(user.UpdatedAt)),
			},
			Country:     &inventory.Country{Id: country.ID, Name: country.Name},
			State:       &inventory.State{Id: state.ID, Name: state.Name},
			Lga:         &inventory.LGA{Id: lga.ID, Name: lga.Name, StateId: lga.StateID},
			Category:    &inventory.CategoryResponse{Id: category.ID, Name: category.Name, Description: category.Description, IconClass: category.IconClass, CategorySlug: category.CategorySlug},
			Subcategory: &inventory.SubCategoryResponse{Id: subcategory.ID, Name: subcategory.Name, Description: subcategory.Description, IconClass: subcategory.IconClass, SubcategorySlug: subcategory.SubCategorySlug},
			Images:      mapToProtoImages(di.Images),
		}, nil

	case err := <-errInventoryExistCh:
		log.Println(fmt.Errorf("error fetching the inventory: %v", err))
		return nil, fmt.Errorf("error fetching the inventory")

	case <-ctx.Done():
		// If the operation timed out, return a timeout error
		return nil, fmt.Errorf("request timed out while fetching user who is been rated")
	}

}

func mapToProtoImages(images []data.InventoryImage) []*inventory.InventoryImage {
	var protoImages []*inventory.InventoryImage
	for _, img := range images {
		protoImages = append(protoImages, &inventory.InventoryImage{
			Id:          img.ID,
			LiveUrl:     img.LiveUrl,
			LocalUrl:    img.LocalUrl,
			InventoryId: img.InventoryId,
			CreatedAt:   timestamppb.New(img.CreatedAt),
			UpdatedAt:   timestamppb.New(img.UpdatedAt),
		})
	}
	return protoImages
}

func (i *InventoryServer) GetUserRatings(ctx context.Context, req *inventory.GetResourceWithIDAndPagination) (*inventory.UserRatingsResponse, error) {

	// Result and error channels
	resultCh := make(chan []*data.UserRating, 1)
	totalRecCh := make(chan int32, 1)
	errCh := make(chan error, 1)

	// Timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		result, totalRow, err := i.Models.GetUserRatings(timeoutCtx, req.Id.Id, req.Pagination.Page, req.Pagination.Limit)
		if err != nil {
			errCh <- err
			return
		}
		log.Printf("Fetched %d rows from database (Total: %d)", len(result), totalRow)
		resultCh <- result
		totalRecCh <- totalRow
	}()

	select {
	case data := <-resultCh:
		var allUserRating []*inventory.UserRatingResponse

		for _, singleRating := range data {
			rating := &inventory.UserRatingResponse{
				Id:             singleRating.ID,
				UserId:         singleRating.UserId,
				RaterId:        singleRating.RaterId,
				Rating:         singleRating.Rating,
				Comment:        singleRating.Comment,
				CreatedAtHuman: formatTimestamp(timestamppb.New(singleRating.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(singleRating.UpdatedAt)),
				Rater: &inventory.User{
					Id:        singleRating.RaterId,
					FirstName: singleRating.RaterDetails.FirstName,
					Email:     singleRating.RaterDetails.Email,
					LastName:  singleRating.RaterDetails.LastName,
				},
			}
			allUserRating = append(allUserRating, rating)
		}

		summary, err := i.Models.GetUserRatingSummary(timeoutCtx, req.Id.Id)
		if err != nil {
			return nil, fmt.Errorf("error retrieving rating summary for user")
		}

		return &inventory.UserRatingsResponse{
			UserRatings: allUserRating,
			PageDetail:  &inventory.PaginationParam{Page: req.Pagination.Page, Limit: req.Pagination.Limit},
			Total:       <-totalRecCh,
			RatingSumary: &inventory.RatingSummary{
				FiveStar:      summary.FiveStar,
				FourStar:      summary.FourStar,
				ThreeStar:     summary.ThreeStar,
				TwoStar:       summary.TwoStar,
				OneStar:       summary.OneStar,
				AverageRating: summary.AverageRating,
			},
		}, nil

	case err := <-errCh:
		log.Println(fmt.Errorf("error fetching user ratings: %v", err))
		return nil, fmt.Errorf("error fetching user ratings")
	case <-ctx.Done():
		return nil, fmt.Errorf("request timed out while fetching user ratings")
	}
}

func (i *InventoryServer) GetInventoryRatings(ctx context.Context, req *inventory.GetResourceWithIDAndPagination) (*inventory.InventoryRatingsResponse, error) {

	// Result and error channels
	resultCh := make(chan []*data.InventoryRating, 1)
	totalRecCh := make(chan int32, 1)
	errCh := make(chan error, 1)

	// Timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		result, totalRow, err := i.Models.GetInventoryRatings(timeoutCtx, req.Id.Id, req.Pagination.Page, req.Pagination.Limit)
		if err != nil {
			errCh <- err
			return
		}
		log.Printf("Fetched %d rows from database (Total: %d)", len(result), totalRow)
		resultCh <- result
		totalRecCh <- totalRow
	}()

	select {
	case data := <-resultCh:
		var allInventoryRating []*inventory.InventoryRatingResponse

		for _, singleRating := range data {
			rating := &inventory.InventoryRatingResponse{
				Id:             singleRating.ID,
				RaterId:        singleRating.RaterId,
				Rating:         singleRating.Rating,
				Comment:        singleRating.Comment,
				CreatedAtHuman: formatTimestamp(timestamppb.New(singleRating.CreatedAt)),
				UpdatedAtHuman: formatTimestamp(timestamppb.New(singleRating.UpdatedAt)),
				Rater: &inventory.User{
					Id:        singleRating.RaterId,
					FirstName: singleRating.RaterDetails.FirstName,
					Email:     singleRating.RaterDetails.Email,
					LastName:  singleRating.RaterDetails.LastName,
				},
			}

			var replies []*inventory.InventoryRatingReplyResponse
			for _, reply := range singleRating.Replies {
				replierDetails := &inventory.User{
					Id:        reply.ReplierDetails.ID,
					FirstName: reply.ReplierDetails.FirstName,
					LastName:  reply.ReplierDetails.LastName,
					Email:     reply.ReplierDetails.Email,
				}

				var parentReplyID string
				if reply.ParentReplyID != nil {
					parentReplyID = *reply.ParentReplyID
				} else {
					parentReplyID = "" // Default value for nil
				}

				replies = append(replies, &inventory.InventoryRatingReplyResponse{
					Id:             reply.ID,
					ParentReplyId:  parentReplyID,
					Comment:        reply.Comment,
					CreatedAtHuman: formatTimestamp(timestamppb.New(reply.CreatedAt)),
					UpdatedAtHuman: formatTimestamp(timestamppb.New(reply.UpdatedAt)),
					Replier:        replierDetails,
				})
			}

			rating.Replies = replies
			allInventoryRating = append(allInventoryRating, rating)
		}

		summary, err := i.Models.GetInventoryRatingSummary(timeoutCtx, req.Id.Id)
		if err != nil {
			log.Println(err, "EROOR FETCHING SUMMARY")
			log.Println(req.Id.Id, "EROOR THE ID")
			return nil, fmt.Errorf("error retrieving rating summary for user")
		}

		return &inventory.InventoryRatingsResponse{
			InventoryRatings: allInventoryRating,
			PageDetail:       &inventory.PaginationParam{Page: req.Pagination.Page, Limit: req.Pagination.Limit},
			Total:            <-totalRecCh,
			RatingSumary: &inventory.RatingSummary{
				FiveStar:      summary.FiveStar,
				FourStar:      summary.FourStar,
				ThreeStar:     summary.ThreeStar,
				TwoStar:       summary.TwoStar,
				OneStar:       summary.OneStar,
				AverageRating: summary.AverageRating,
			},
		}, nil

	case err := <-errCh:
		log.Println(fmt.Errorf("error fetching inventory ratings: %v", err))
		return nil, fmt.Errorf("error fetching inventory ratings")
	case <-ctx.Done():
		return nil, fmt.Errorf("request timed out while fetching inventory ratings")
	}
}

func (i *InventoryServer) ReplyInventoryRating(ctx context.Context, req *inventory.ReplyToRatingRequest) (*inventory.ReplyToRatingResponse, error) {
	// Result and error channels
	resultCh := make(chan *data.InventoryRatingReply, 1)
	errCh := make(chan error, 1)

	// Timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	param := &data.ReplyRatingPayload{
		RatingID:      req.RatingId,
		ReplierID:     req.ReplierId,
		Comment:       req.Comment,
		ParentReplyID: req.ParentReplyId,
	}

	go func(param *data.ReplyRatingPayload) {
		result, err := i.Models.CreateInventoryRatingReply(timeoutCtx, param)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}(param)

	select {
	case data := <-resultCh:

		var parentReplyID string
		if data.ParentReplyID != nil {
			parentReplyID = *data.ParentReplyID
		} else {
			parentReplyID = "" // Default value for nil
		}

		return &inventory.ReplyToRatingResponse{
			Id:             data.ID,
			RatingId:       data.RatingID,
			ReplierId:      data.ReplierID,
			ParentReplyId:  parentReplyID,
			Comment:        data.Comment,
			CreatedAtHuman: formatTimestamp(timestamppb.New(data.CreatedAt)),
			UpdatedAtHuman: formatTimestamp(timestamppb.New(data.UpdatedAt)),
		}, nil

	case err := <-errCh:
		log.Println(fmt.Errorf("error fetching inventory ratings: %v", err))
		return nil, fmt.Errorf("error fetching inventory ratings")
	case <-ctx.Done():
		return nil, fmt.Errorf("request timed out while fetching inventory ratings")
	}
}

func (i *InventoryServer) ReplyUserRating(ctx context.Context, req *inventory.ReplyToRatingRequest) (*inventory.ReplyToRatingResponse, error) {
	// Result and error channels
	resultCh := make(chan *data.UserRatingReply, 1)
	errCh := make(chan error, 1)

	// Timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	param := &data.ReplyRatingPayload{
		RatingID:      req.RatingId,
		ReplierID:     req.ReplierId,
		Comment:       req.Comment,
		ParentReplyID: req.ParentReplyId,
	}

	go func(param *data.ReplyRatingPayload) {
		result, err := i.Models.CreateUserRatingReply(timeoutCtx, param)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}(param)

	select {
	case data := <-resultCh:

		var parentReplyID string
		if data.ParentReplyID != nil {
			parentReplyID = *data.ParentReplyID
		} else {
			parentReplyID = "" // Default value for nil
		}

		return &inventory.ReplyToRatingResponse{
			Id:             data.ID,
			RatingId:       data.RatingID,
			ReplierId:      data.ReplierID,
			ParentReplyId:  parentReplyID,
			Comment:        data.Comment,
			CreatedAtHuman: formatTimestamp(timestamppb.New(data.CreatedAt)),
			UpdatedAtHuman: formatTimestamp(timestamppb.New(data.UpdatedAt)),
		}, nil

	case err := <-errCh:
		log.Println(fmt.Errorf("error fetching inventory ratings: %v", err))
		return nil, fmt.Errorf("error fetching inventory ratings")
	case <-ctx.Done():
		return nil, fmt.Errorf("request timed out while fetching inventory ratings")
	}
}

const handlerTimeout = 10 * time.Second

func (s *InventoryServer) SearchInventory(
	ctx context.Context,
	req *inventory.SearchInventoryRequest,
) (*inventory.InventoryCollection, error) {

	// 1) Derive a timeout’d context
	ctx, cancel := context.WithTimeout(ctx, handlerTimeout)
	defer cancel()

	// 2) Build your data.SearchPayload (Limit/Offset as strings)
	param := &data.SearchPayload{
		CountryID:       req.CountryId,
		StateID:         req.StateId,
		LgaID:           req.LgaId,
		Text:            req.Text,
		Limit:           req.Limit,
		Offset:          req.Offset,
		CategoryID:      req.CategoryId,
		SubcategoryID:   req.SubcategoryId,
		Ulid:            req.Ulid,
		StateSlug:       req.StateSlug,
		CountrySlug:     req.CountrySlug,
		LgaSlug:         req.LgaSlug,
		CategorySlug:    req.CategorySlug,
		SubcategorySlug: req.SubcategorySlug,
	}

	// 3) Call your repo
	dc, err := s.Models.SearchInventory(ctx, param)
	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"failed to search inventories: %v", err)
	}

	// 4) Map data.InventoryCollection → proto.InventoryCollection
	resp := &inventory.InventoryCollection{
		Inventories: []*inventory.Inventory{}, // <- explicitly set this
		TotalCount:  dc.TotalCount,
		Offset:      dc.Offset,
		Limit:       dc.Limit,
	}
	for _, di := range dc.Inventories {
		inv := &inventory.Inventory{
			Id:             di.Id,
			Name:           di.Name,
			Description:    di.Description,
			UserId:         di.UserId,
			CategoryId:     di.CategoryId,
			SubcategoryId:  di.SubcategoryId,
			Promoted:       di.Promoted,
			Deactivated:    di.Deactivated,
			CreatedAt:      di.CreatedAt,
			UpdatedAt:      di.UpdatedAt,
			CreatedAtHuman: formatTimestamp(di.CreatedAt),
			UpdatedAtHuman: formatTimestamp(di.UpdatedAt),
			Slug:           di.Slug,
			Ulid:           di.Ulid,
			OfferPrice:     di.OfferPrice,

			ProductPurpose:  di.ProductPurpose,
			Quantity:        di.Quantity,
			IsAvailable:     di.IsAvailable,
			RentalDuration:  di.RentalDuration,
			SecurityDeposit: di.SecurityDeposit,
			Metadata:        di.Metadata,
			Negotiable:      di.Negotiable,
			PrimaryImage:    di.PrimaryImage,
			MinimumPrice:    di.MinimumPrice,

			CountryId: di.CountryId,
			Country:   &inventory.Country{Id: di.CountryId, Name: di.Country.Name},
			StateId:   di.StateId,
			State:     &inventory.State{Id: di.StateId, Name: di.State.Name, Code: di.State.Code, CountryId: di.State.CountryId},
			LgaId:     di.LgaId,
			Lga:       &inventory.LGA{Id: di.LgaId, Name: di.Lga.Name, StateId: di.Lga.StateId},
			Images:    make([]*inventory.InventoryImage, len(di.Images)),
			User:      &inventory.User{Id: di.User.Id, FirstName: di.User.FirstName, Email: di.User.Email, LastName: di.User.LastName, Phone: di.User.Phone},

			StateSlug:       di.StateSlug,
			CountrySlug:     di.CountrySlug,
			LgaSlug:         di.LgaSlug,
			CategorySlug:    di.CategorySlug,
			SubcategorySlug: di.SubcategorySlug,
		}
		// map images
		for i, img := range di.Images {
			inv.Images[i] = &inventory.InventoryImage{
				Id:          img.Id,
				LiveUrl:     img.LiveUrl,
				LocalUrl:    img.LocalUrl,
				InventoryId: img.InventoryId,
				CreatedAt:   timestamppb.New(img.CreatedAt.AsTime()),
				UpdatedAt:   timestamppb.New(img.CreatedAt.AsTime()),
			}
		}
		resp.Inventories = append(resp.Inventories, inv)
	}

	return resp, nil
}
