# FROM --platform=linux/amd64 alpine:latest

# RUN mkdir /app

# COPY inventoryApp /app

# CMD [ "/app/inventoryApp" ]


# Use Go 1.22 for building and running tests
FROM golang:1.22-alpine AS builder

# Set the working directory in the container
WORKDIR /app

# Install make and other necessary packages (e.g., git)
RUN apk add --no-cache git make

# Copy the entire source code into the container
COPY . .

# Download the Go modules dependencies
RUN go mod download

# Default command to run the binary
CMD ["/app/inventoryApp"]