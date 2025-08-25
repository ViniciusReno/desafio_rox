# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o api ./cmd/api

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/api /app/api
EXPOSE 8080
CMD ["/app/api"]
