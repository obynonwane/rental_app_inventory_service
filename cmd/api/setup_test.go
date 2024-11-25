package main

import (
	"os"
	"testing"

	"github.com/obynonwane/inventory-service/data"
)

var testApp Config

// Special function to setup and tear down testing environments
// It takes a single argument of type *testing.M
func TestMain(m *testing.M) {

	repo := data.NewPostgresTestRepository(nil)

	testApp.Repo = repo
	//execute the tests and benchmarks.
	//It returns an exit code that indicates
	//whether the tests passed or failed
	os.Exit(m.Run()) // run all of my test
}
