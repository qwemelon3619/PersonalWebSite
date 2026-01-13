# syntax=docker/dockerfile:1

# --- Development (air) image ---
FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY . .
WORKDIR /app/services/web-front
RUN go install github.com/air-verse/air@latest
EXPOSE 3000
ENV GIN_MODE=debug
CMD ["air", "-c", ".air.toml"]

# --- Production build ---
FROM golang:1.25-alpine AS prod-build
WORKDIR /app
COPY . .
WORKDIR /app/services/web-front
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o web-front ./cmd/main.go

# --- Production image ---
FROM alpine:latest AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app/services/web-front
COPY --from=prod-build /app/services/web-front/web-front .
EXPOSE 3000
ENV GIN_MODE=release
CMD ["./web-front"]
