
# --- Development (air) image ---
FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY . .
WORKDIR /app/services/auth-service
RUN go install github.com/air-verse/air@latest
EXPOSE 8081
ENV GIN_MODE=debug
CMD ["air", "-c", ".air.toml"]

# --- Production build ---
FROM golang:1.25-alpine AS prod-build
WORKDIR /app
COPY . .
WORKDIR /app/services/auth-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/main.go

# --- Production image ---
FROM alpine:latest AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=prod-build /app/services/auth-service/auth-service .
EXPOSE 8081
ENV GIN_MODE=release
CMD ["./auth-service"]
