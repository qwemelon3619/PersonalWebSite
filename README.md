

# Personal Blog MSA

A Go-based microservices blog platform featuring multilingual support, automatic translation, and cloud-native architecture.

## Architecture

This project follows a microservices architecture with clean separation of concerns, enabling independent scaling and deployment of each service.

### Services

- **API Gateway** (`services/api-gateway`): Single entry point using Gin framework. Handles JWT token validation, request routing, and proxies to backend services.
- **Auth Service** (`services/auth-service`): Manages user authentication, registration, login, JWT token generation/refresh, and Redis-backed token blacklisting.
- **Post Service** (`services/post-service`): Handles blog post CRUD operations, tag management, and integrates with translation and image services.
- **Image Service** (`services/img-service`): Dedicated image upload service that stores files in Azure Blob Storage (Azurite for local development).
- **Web Front** (`services/web-front`): Server-rendered web interface using Go HTML templates, providing the public blog website.

### Shared Components

- **Shared Packages** (`pkg/`): Reusable components including JWT utilities, middleware, database connections, configuration management, and logging.
- **Infrastructure**: Docker Compose orchestration with PostgreSQL database, Redis cache, and Azurite blob storage emulator.

## Features

### Core Functionality

- **User Authentication**: JWT-based login/registration with secure token management
- **Blog Management**: Full CRUD operations for blog posts with rich text editing (Quill.js)
- **Image Handling**: Seamless image uploads integrated into blog content
- **Tag System**: Post categorization and filtering by tags

### Advanced Features

- **Multilingual Support**: Automatic translation between Korean and English using DeepL API
- **Dynamic Content Rendering**: Client-side language switching with preserved formatting
- **Responsive UI**: Bootstrap-based design with table of contents generation
- **Cloud Storage**: Azure Blob Storage integration for scalable media hosting

### Technical Highlights

- **Clean Architecture**: Each service follows domain-driven design with clear separation of handlers, services, and repositories
- **Microservices Communication**: HTTP-based inter-service communication via API Gateway
- **Containerization**: Full Docker support for consistent development and deployment environments
- **Scalability**: Stateless services with externalized state management

## Technology Stack

- **Backend**: Go 1.21+, Gin web framework
- **Database**: PostgreSQL for persistent data, Redis for caching and token management
- **Storage**: Azure Blob Storage (Azurite for local development)
- **Frontend**: Server-side rendered HTML with Bootstrap CSS, Quill.js for rich text editing
- **Infrastructure**: Docker & Docker Compose for container orchestration
- **External APIs**: DeepL for machine translation

## Quick Start

1. Ensure Docker and Docker Compose are installed
2. Clone the repository and navigate to the project root
3. Configure environment variables (see `docker-compose.yml` for required variables)
4. Run the application:

```bash
docker-compose up --build
```

5. Access the blog at `http://localhost:3000`

## Project Structure

```
├── services/
│   ├── api-gateway/     # API Gateway service
│   ├── auth-service/    # Authentication service
│   ├── post-service/    # Blog post management
│   ├── img-service/     # Image upload service
│   └── web-front/       # Web frontend
├── pkg/                 # Shared packages
├── docker-compose.yml   # Container orchestration
└── README.md
```

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

