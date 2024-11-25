package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/obynonwane/rental-service-proto/inventory"

	"github.com/obynonwane/inventory-service/data"
)

func TestRateUser_WithTestRepository(t *testing.T) {
	// Initialize the test repository
	testRepo := data.NewPostgresTestRepository(nil)

	// Create an instance of the InventoryServer
	server := &InventoryServer{
		Models: testRepo,
	}

	// Define input and expected output
	req := &inventory.UserRatingRequest{
		UserId:  "01197718-a7a9-4af8-9870-661e17cd0d81", // Matches the test data
		RaterId: "7a937e9d-1dc2-4e6d-ba38-d1648b05730c",
		Rating:  5,
		Comment: "Excellent service",
	}

	// Call the gRPC method
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.RateUser(ctx, req)

	// Validate the response
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "6a7b83f0-30cb-4854-a32e-3576bf491858", resp.Id) // Matches the mock response ID
	assert.Equal(t, req.UserId, resp.UserId)
	assert.Equal(t, req.RaterId, resp.RaterId)
	assert.Equal(t, req.Rating, resp.Rating)
	assert.Equal(t, req.Comment, resp.Comment)
}

const bufSize = 1024 * 1024

func TestRateUser_Integration_WithTestRepository(t *testing.T) {
	// Set up Bufconn listener
	listener := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	// Use the test repository
	testRepo := data.NewPostgresTestRepository(nil)
	server := &InventoryServer{
		Models: testRepo,
	}

	// Register the server
	inventory.RegisterInventoryServiceServer(grpcServer, server)

	// Error channel to capture errors from the goroutine
	errCh := make(chan error, 1)

	// Run the server in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			errCh <- fmt.Errorf("Server exited with error: %v", err)
		}
		close(errCh) // Ensure the channel is closed when done
	}()

	// Set up Bufconn Dialer
	bufDialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return listener.Dial()
	}

	// Check for server startup errors
	select {
	case err := <-errCh:
		require.NoError(t, err, "Server failed to start")
	default:
		// No errors; proceed
	}

	// Create a gRPC client connection
	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	client := inventory.NewInventoryServiceClient(conn)

	// Simulate a client request
	req := &inventory.UserRatingRequest{
		UserId:  "01197718-a7a9-4af8-9870-661e17cd0d81",
		RaterId: "7a937e9d-1dc2-4e6d-ba38-d1648b05730c",
		Rating:  5,
		Comment: "Excellent service",
	}

	// Call the gRPC method
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RateUser(ctx, req)

	// Validate the response
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "6a7b83f0-30cb-4854-a32e-3576bf491858", resp.Id) // Matches the mock response ID
	assert.Equal(t, req.UserId, resp.UserId)
	assert.Equal(t, req.RaterId, resp.RaterId)
	assert.Equal(t, req.Rating, resp.Rating)
	assert.Equal(t, req.Comment, resp.Comment)
}
