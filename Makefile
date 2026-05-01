DATABASE_URL ?= postgres://auction:password@localhost:5432/auction_db?sslmode=disable
AMQP_URL ?= amqp://rabbit:password@localhost:5672/
PORT ?= 8082

.DEFAULT_GOAL := help

.PHONY: help build-ui run-server run docker-build test

help: ## Show available Make targets.
	@printf "Auction Core Make targets:\n\n"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run the Go test suite.
	go test ./...

build-ui: ## Install UI dependencies and build the React app.
	cd ui && npm install && npm run build

run-server: ## Run the Go server with local development defaults.
	DATABASE_URL="$(DATABASE_URL)" AMQP_URL="$(AMQP_URL)" PORT="$(PORT)" go run ./cmd/server

run: build-ui run-server ## Build the UI, then run the Go server.

docker-build: ## Build the Docker image.
	docker build -t auction-core .
