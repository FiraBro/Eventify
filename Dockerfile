# Use official Go image
FROM golang:1.25-alpine

# Install required tools
RUN apk add --no-cache bash git

# Install Air with a Go version compatible release
RUN go install github.com/air-verse/air@v1.61.1

# Install migrate CLI with postgres driver support
RUN go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Add Go bin to PATH
ENV PATH=$PATH:/go/bin

# Set working directory
WORKDIR /app

# Copy go.mod & go.sum first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project
COPY . .

# Make entrypoint executable
RUN chmod +x ./docker-entrypoint.sh

# Expose API port
EXPOSE 8080

# Run migrations, then start the app with auto-reload
CMD ["/bin/sh", "-c", "/app/docker-entrypoint.sh"]
