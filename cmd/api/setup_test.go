package main

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/obynonwane/inventory-service/data"
)

var testApp Config

// Special function to setup and tear down testing environments
// It takes a single argument of type *testing.M
func TestMain(m *testing.M) {

	err := godotenv.Load(".env.test")
	if err != nil {
		log.Fatalf("Error loading .env.test file")
	}

	repo := data.NewPostgresTestRepository(nil)

	testApp.Repo = repo
	//execute the tests and benchmarks.
	//It returns an exit code that indicates
	//whether the tests passed or failed
	os.Exit(m.Run()) // run all of my test
}
