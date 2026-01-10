# Build variables
DOCKER_REGISTRY ?= myregistry.azurecr.io
VERSION ?= latest

# Go variables
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# Service names
API_GATEWAY = api-gateway
AUTH_SERVICE = auth-service
POST_SERVICE = post-service

.PHONY: help build test clean docker-build docker-push deploy

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Build commands
build: build-api-gateway build-auth-service build-post-service ## Build all services

build-api-gateway: ## Build API Gateway
	cd services/$(API_GATEWAY) && $(GOBUILD) -o ../../bin/$(API_GATEWAY) ./cmd/main.go

build-auth-service: ## Build Auth Service
	cd services/$(AUTH_SERVICE) && $(GOBUILD) -o ../../bin/$(AUTH_SERVICE) ./cmd/main.go

build-post-service: ## Build Post Service
	cd services/$(POST_SERVICE) && $(GOBUILD) -o ../../bin/$(POST_SERVICE) ./cmd/main.go

# Test commands
test: ## Run tests for all services
	$(GOTEST) ./...

test-auth-service: ## Test Auth Service
	cd services/$(AUTH_SERVICE) && $(GOTEST) ./...

test-post-service: ## Test Post Service
	cd services/$(POST_SERVICE) && $(GOTEST) ./...

test-api-gateway: ## Test API Gateway
	cd services/$(API_GATEWAY) && $(GOTEST) ./...

test-integration: ## Run integration tests
	$(GOTEST) -tags=integration ./...

# Docker commands
docker-build: docker-build-api-gateway docker-build-auth-service docker-build-post-service ## Build all Docker images

docker-build-api-gateway: ## Build API Gateway Docker image
	docker build -f services/$(API_GATEWAY)/Dockerfile -t $(DOCKER_REGISTRY)/$(API_GATEWAY):$(VERSION) .

docker-build-auth-service: ## Build Auth Service Docker image
	docker build -f services/$(AUTH_SERVICE)/Dockerfile -t $(DOCKER_REGISTRY)/$(AUTH_SERVICE):$(VERSION) .

docker-build-post-service: ## Build Post Service Docker image
	docker build -f services/$(POST_SERVICE)/Dockerfile -t $(DOCKER_REGISTRY)/$(POST_SERVICE):$(VERSION) .

docker-push: ## Push all Docker images to registry
	docker push $(DOCKER_REGISTRY)/$(API_GATEWAY):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(AUTH_SERVICE):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(POST_SERVICE):$(VERSION)

# Development commands
dev-up: ## Start development environment
	docker-compose up -d

dev-down: ## Stop development environment
	docker-compose down

dev-logs: ## Show logs from development environment
	docker-compose logs -f

# Utility commands
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf bin/

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format code
	$(GOCMD) fmt ./...

lint: ## Run linter
	golangci-lint run

# Deployment commands
deploy-azure: ## Deploy to Azure Container Apps
	cd deployments/azure && az deployment group create \
		--resource-group personal-blog-rg \
		--template-file main.bicep \
		--parameters @parameters.json