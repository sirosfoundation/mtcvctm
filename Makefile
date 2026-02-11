.PHONY: build test coverage clean docker-build docker-push lint fmt help install

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X github.com/sirosfoundation/mtcvctm/cmd/mtcvctm/cmd.Version=$(VERSION) -X github.com/sirosfoundation/mtcvctm/cmd/mtcvctm/cmd.Commit=$(COMMIT)"

# Docker variables
DOCKER_IMAGE ?= ghcr.io/sirosfoundation/mtcvctm
DOCKER_TAG ?= $(VERSION)

# Go variables
GOBIN ?= $(shell go env GOPATH)/bin

# Default target
all: build

## build: Build the mtcvctm binary
build:
	go build $(LDFLAGS) -o mtcvctm ./cmd/mtcvctm

## install: Install mtcvctm to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/mtcvctm

## test: Run tests
test:
	go test -v -race ./...

## coverage: Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "Coverage report generated: coverage.out"
	@echo "To view HTML report: go tool cover -html=coverage.out"

## coverage-html: Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "HTML coverage report: coverage.html"

## lint: Run linters
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format code
fmt:
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy go modules
tidy:
	go mod tidy

## clean: Clean build artifacts
clean:
	rm -f mtcvctm
	rm -f coverage.out coverage.html
	rm -rf dist/
	rm -f *.vctm
	rm -f examples/*.vctm
	rm -f __debug_bin*
	rm -rf vendor/

## docker-build: Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

## docker-push: Push Docker image
docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

## docker-run: Run Docker container
docker-run:
	docker run --rm -v $(PWD):/workspace -w /workspace $(DOCKER_IMAGE):latest convert example.md

## example: Run example conversion
example: build
	@echo "Creating example markdown file..."
	@mkdir -p examples/images
	@echo '---' > examples/identity.md
	@echo 'vct: https://example.com/credentials/identity' >> examples/identity.md
	@echo 'background_color: "#1a365d"' >> examples/identity.md
	@echo 'text_color: "#ffffff"' >> examples/identity.md
	@echo '---' >> examples/identity.md
	@echo '' >> examples/identity.md
	@echo '# Identity Credential' >> examples/identity.md
	@echo '' >> examples/identity.md
	@echo 'A verifiable credential for identity verification.' >> examples/identity.md
	@echo '' >> examples/identity.md
	@echo '## Claims' >> examples/identity.md
	@echo '' >> examples/identity.md
	@echo '- `given_name` (string): The given name of the holder [mandatory]' >> examples/identity.md
	@echo '- `family_name` (string): The family name of the holder [mandatory]' >> examples/identity.md
	@echo '- `birth_date` (date): Date of birth [sd=always]' >> examples/identity.md
	@echo '- `nationality` (string): Nationality of the holder' >> examples/identity.md
	@echo "" 
	@echo "Converting to VCTM..."
	./mtcvctm convert examples/identity.md --base-url https://registry.example.com
	@echo ""
	@echo "Generated VCTM:"
	@cat examples/identity.vctm

## help: Show this help
help:
	@echo "mtcvctm - Markdown To Create Verifiable Credential Type Metadata"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
