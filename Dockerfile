FROM golang:1.24-alpine

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Expose the port
EXPOSE 8082

# Run the binary
CMD ["./main"]
