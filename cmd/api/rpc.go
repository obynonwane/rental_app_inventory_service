package main

import (
	"context"
	"time"

	"github.com/obynonwane/inventory-service/data"
)

// 1. setup a type - server type
type RPCServer struct {
	App *Config
}

// 3. write methods we want to expose via RPC
func (r *RPCServer) RetrieveUsers(_ struct{}, resp *([]*data.User)) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	defer cancel()

	users, err := r.App.Repo.GetAll(ctx)

	if err != nil {
		return err
	}
	// Assign users to *resp to set the response
	*resp = users

	return nil

}
