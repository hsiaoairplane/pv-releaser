# Use a minimal base image with Go installed
FROM golang:1.25 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go source code
COPY . .

# Build the Go binary
RUN go build -o pv-releaser main.go

# Run the webhook
CMD ["/app/pv-releaser"]
