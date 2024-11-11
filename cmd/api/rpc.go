package main

import (
	"github.com/obynonwane/inventory-service/data"
)

// 1. setup a type - server type
type RPCServer struct {
	App *Config
}

// 2. define the kind of payload you would receive from RPC - payload type
type RPCPayload struct {
	Name string
	Data string
}

// 3. write methods we want to expose via RPC
func (r *RPCServer) RetrieveUsers(_ struct{}, resp *([]*data.User)) error {

	users, err := r.App.Repo.GetAll()

	if err != nil {
		return err
	}
	// Assign users to *resp to set the response
	*resp = users

	return nil

}
