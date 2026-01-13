

# --- Development (air) image ---
FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY . .
WORKDIR /app/services/post-service
RUN go install github.com/air-verse/air@latest
EXPOSE 8082
ENV GIN_MODE=debug
CMD ["air", "-c", ".air.toml"]


FROM golang:1.25-alpine AS prod-build
WORKDIR /app
# Copy the entire project (to preserve go.work and all service/pkg paths)
COPY . .
# Build
WORKDIR /app/services/post-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/post-service ./cmd/main.go


FROM alpine:latest AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=prod-build /app/post-service .
EXPOSE 8082
CMD ["./post-service"]