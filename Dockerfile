# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server .

# Run stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 4100
CMD ["./server"]
