# syntax=docker/dockerfile:1

# --- Development (air) image ---
FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY . .
WORKDIR /app/services/api-gateway
RUN go install github.com/air-verse/air@latest
EXPOSE 8080
ENV GIN_MODE=debug
CMD ["air", "-c", ".air.toml"]

# --- Production build ---
FROM golang:1.25-alpine AS prod-build
WORKDIR /app
COPY . .
WORKDIR /app/services/api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway ./cmd/main.go

# --- Production image ---
FROM alpine:latest AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=prod-build /app/services/api-gateway/api-gateway .
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./api-gateway"]
