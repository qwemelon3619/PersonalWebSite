

# Personal Blog MSA - Project Overview

This project is a personal blog platform built with a Go-based microservices architecture (MSA). Each service (authentication, posts, API gateway) operates independently, with support for Docker Compose and Azure deployment.

## Architecture

- **API Gateway**: The single entry point for all external requests. Proxies to auth/post services and applies JWT authentication middleware.
- **Auth Service**: Handles JWT-based authentication, registration/login, token refresh, and blacklist management.
- **Post Service**: Handles CRUD operations for blog posts.
- **Shared Package (pkg)**: Reusable code for JWT, middleware, DB, config, etc.
- **DB/Redis**: Uses PostgreSQL and Redis containers.

## Tech Stack
- Go 1.25+
- Gin Web Framework
- PostgreSQL, Redis
- Docker, Docker Compose
- Azure Container Apps (deployment supported)

## Development & Usage

### 1. Environment Variables & Config
- All services use environment variable-based configuration.
- See `pkg/config/config.go` and `docker-compose.yml` for key variables.

### 2. Local Development
```bash
# Start all services and infrastructure
docker-compose up --build

# Develop individual services
cd services/auth-service && go run cmd/main.go
cd services/post-service && go run cmd/main.go
cd services/api-gateway && go run cmd/main.go
```

### 3. API Endpoints
- `/api/v1/auth/login`, `/api/v1/auth/register`, `/api/v1/auth/refresh` (via API Gateway)
- `/api/v1/posts`, `/api/v1/posts/:id` (via API Gateway)

### 4. Auth/Token Flow
- JWT access tokens are validated only in the Gateway middleware
- Refresh tokens are validated and blacklisted only in the auth-service (using Redis)
- When the access token expires, the client requests `/auth/refresh` with the refresh token
- If the refresh token is expired, the user must log in again

### 5. Security & Networking
- In production, only the API Gateway is exposed externally
- Auth/post services are accessible only within the internal network

### 6. Deployment
- Azure Bicep template provided (`deployments/azure/main.bicep`)
- Each service is built and deployed as a container image

## Folder Structure
```
├── api/openapi/           # OpenAPI specs
├── deployments/           # Azure, Docker deployment scripts
├── pkg/                   # Shared packages (jwt, config, middleware, etc.)
├── services/
│   ├── api-gateway/
│   ├── auth-service/
│   └── post-service/
└── docker-compose.yml
```

## Additional Notes
- Each service follows Clean Architecture (handler-service-repository-domain)
- Designed for easy testing, scaling, and production deployment

---
Contact: [Project Owner]

## Contributing

Please read the [coding guidelines](.github/copilot-instructions.md) for AI development assistance.