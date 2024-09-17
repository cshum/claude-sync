# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=claudesync
BINARY_UNIX=$(BINARY_NAME)_unix

# Main package path
MAIN_PACKAGE=./cmd/claudesync

# Build the project
all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)
	./$(BINARY_NAME)

# Test the project
test:
	$(GOTEST) -v ./...

# Clean the binary
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run all the tests and generate a coverage report
cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Install all the required dependencies
deps:
	$(GOGET) ./...

# Update dependencies
update-deps:
	$(GOGET) -u ./...

# Tidy up the go.mod file
tidy:
	$(GOMOD) tidy

# Run golangci-lint
lint:
	golangci-lint run

# Cross compilation for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PACKAGE)

# Install golangci-lint
install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.52.2

.PHONY: all build run test clean cover deps update-deps tidy lint build-linux install-lint
