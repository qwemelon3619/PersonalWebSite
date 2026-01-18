

# Personal Blog MSA - Project Overview

This repository contains a Go-based microservices blog platform designed for local development and deployment to Azure. It uses a small set of services (gateway, auth, posts, image uploader, web front) following a Clean Architecture pattern.

## Architecture

- **API Gateway**: The single entry point for all external requests. Proxies to auth/post services and applies JWT authentication middleware.
- **Auth Service**: Handles JWT-based authentication, registration/login, token refresh, and blacklist management.
- **Post Service**: Handles CRUD operations for blog posts.
- **Shared Package (pkg)**: Reusable code for JWT, middleware, DB, config, etc.
- **DB/Redis**: Uses PostgreSQL and Redis containers.
# Personal Blog MSA - Project Overview

This repository contains a Go-based microservices blog platform designed for local development and deployment to Azure. It follows Clean Architecture and is split into small services to keep responsibilities clear.

Overview of services

- api-gateway: Gin-based gateway that validates access tokens and proxies requests to backend services.
- auth-service: Login, register, refresh tokens; Redis-backed refresh blacklist.
- post-service: Blog CRUD and image upload helper (uploads to Azure Blob Storage).
- web-front: Server-rendered templates and static assets (public website).

Tech

- Go 1.25+, Gin
- PostgreSQL, Redis
- Docker / Docker Compose
- Azure Blob Storage (Azurite for local dev, real account for prod)

Quick start (local)

1. Ensure Docker is running.

2. Provide environment variables used by services (examples in `docker-compose.yml` and each service `internal/config`):

	- `AZURE_STORAGE_ACCOUNT`, `AZURE_STORAGE_ACCESS_KEY`, `AZURE_BLOB_CONTAINER` (image uploads)
	- `POSTGRE_DB_URL`, `POSTGRE_DB_PORT`, `POSTGRE_DB_USER`, `POSTGRE_DB_PASSWORD`, `POSTGRE_DB_NAME`
	- `REDIS_DB_URL`, `REDIS_DB_PORT`, `REDIS_DB_PASSWORD`
	- `JWT_SECRET_KEY`

3. Start all services (recommended):

```bash
docker-compose up --build
```

4. Access the web front at `http://localhost:3000` and the API Gateway at `http://localhost:8080`.

Hot-reload for web-front (dev)

- The `web-front` service supports hot-reload using `air`. The dev `Dockerfile` installs `air` and `.air.toml` watches templates, static assets, and Go files.
- To run only `web-front` in dev mode (with hot-reload):

```bash
AIR_DEV=1 docker-compose up --build web-front
```

Development notes

- All services use env-based config. See `pkg/config/config.go` and each service's `internal/config` for required keys.
- Run services individually for focused development:

```bash
cd services/auth-service && go run cmd/main.go
cd services/post-service && go run cmd/main.go
cd services/api-gateway && go run cmd/main.go
```

API endpoints (gateway)

- `/api/v1/auth/*` → auth-service (login/register/refresh)
- `/api/v1/posts*` → post-service (CRUD)

Auth / Token flow

- Access tokens are validated by the API Gateway middleware. Write routes require a valid access token.
- Refresh tokens and blacklist logic are handled by `auth-service` using Redis. When access tokens expire, clients call `/api/v1/auth/refresh`.

Image upload (Azure Blob Storage)

- `post-service` exposes `POST /api/v1/upload-image` which accepts JSON: `filename`, `data` (base64 data URI or pure base64 string), and `mimeType`.
- The handler decodes base64 and uploads the blob to the configured container, returning a URL.
- For local testing you can use Azurite or a real Azure Storage account.

Example curl (base64 upload):

```bash
DATA=$(base64 -w 0 local.jpg)
curl -X POST http://localhost:8082/api/v1/upload-image \
	-H "Content-Type: application/json" \
	-d '{"filename":"local.jpg","data":"data:image/jpeg;base64,'"$DATA"'","mimeType":"image/jpeg"}'
```

CI / Image tagging

- The GitHub Actions workflow uses `buildx` and tags images with the commit SHA by default.
- If the workflow is triggered by a tag push, it uses the Git tag as the image tag instead of SHA.

Useful commands

- Build a single service locally:
	```bash
	cd services/web-front && go run cmd/main.go
	```
- Run only web-front with hot reload (dev):
	```bash
	AIR_DEV=1 docker-compose up --build web-front
	```

Tips & recommendations

- For production on Azure, prefer Managed Identity or Service Principal instead of Shared Key.
- Keep secrets out of `docker-compose.yml`; use CI/CD or platform-managed secrets.
- If `web-front` fails to load templates in container, ensure the working directory and `GIN_MODE` align with the template paths.

Contact

Seung Pyo Lee (lspyo11@gmail.com)

