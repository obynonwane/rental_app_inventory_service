
INVENTORY_BINARY=inventoryApp

# build_inventory_service: builds the inventory binary as a linux executable
build_inventory_service: ## Build the inventory service binary
	@echo "Building inventory service binary..."
	@cd cmd/api && env GOOS=linux CGO_ENABLED=0 go build -o ../../${INVENTORY_BINARY} 
	@echo "Done!"

test: 
	cd cmd/api && go test -v -cover ./...

