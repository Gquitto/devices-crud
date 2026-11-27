BINARY  := bin/api
PKG     := ./cmd/api
COVER   := coverage.out
# Count coverage across package boundaries (e.g. the SQLite tests exercising
# device.Restore), excluding the devicetest test double. cmd/api is wiring only.
COVERPKG := $(shell go list ./internal/... | grep -v /devicetest | paste -sd, -)
BASE_URL ?= http://localhost:8080

.DEFAULT_GOAL := help

.PHONY: help run build test cover cover-html insomnia insomnia-verify vet fmt tidy docker-build up down logs clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## Run the API locally
	go run $(PKG)

build: ## Build the binary into bin/
	CGO_ENABLED=0 go build -trimpath -o $(BINARY) $(PKG)

test: ## Run all tests with the race detector
	go test -race -count=1 ./...

cover: ## Run tests with coverage and print the summary
	go test -count=1 -covermode=atomic -coverpkg=$(COVERPKG) -coverprofile=$(COVER) ./...
	go tool cover -func=$(COVER) | tail -n 1

insomnia: ## Regenerate api/insomnia.json from scripts/insomnia.py
	python3 scripts/insomnia.py api/insomnia.json

insomnia-verify: ## Run the Insomnia scenarios against a running API (BASE_URL=...)
	python3 scripts/insomnia.py --verify $(BASE_URL)

cover-html: cover ## Open the coverage report in the browser
	go tool cover -html=$(COVER)

vet: ## Run go vet
	go vet ./...

fmt: ## Format the code
	gofmt -s -w .

tidy: ## Tidy go.mod/go.sum
	go mod tidy

docker-build: ## Build the Docker image
	docker compose build

up: ## Start the stack with Docker Compose
	docker compose up --build -d

down: ## Stop the stack
	docker compose down

logs: ## Follow API container logs
	docker compose logs -f api

clean: ## Remove build and coverage artifacts
	rm -rf bin $(COVER) coverage.html
