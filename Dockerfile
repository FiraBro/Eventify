# Use official Go image
FROM golang:1.25-alpine

# Install required tools
RUN apk add --no-cache bash git make

# Install Air with a Go version compatible release
RUN go install github.com/air-verse/air@v1.61.1

# Install migrate CLI with postgres driver support
RUN go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install swag CLI for generating Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Add Go bin to PATH
ENV PATH=$PATH:/go/bin

# Make Go downloads more tolerant of an unstable network:
# - HTTP/1.1 instead of HTTP/2 (HTTP/2 streams were being dropped)
# - fall back to fetching directly from source if the proxy fails
ENV GODEBUG=http2client=0
ENV GOPROXY=https://proxy.golang.org,direct

# Set working directory
WORKDIR /app

# Copy go.mod & go.sum first (for caching)
COPY go.mod go.sum ./

# Retry the download; modules already fetched are kept between attempts
RUN for i in 1 2 3 4 5 6; do \
      go mod download && exit 0; \
      echo "go mod download failed (attempt $i), retrying..."; \
      sleep 5; \
    done; exit 1

# Copy the rest of the project
COPY . .

# Make entrypoint executable
RUN chmod +x ./docker-entrypoint.sh

# Expose API port
EXPOSE 8080

# Run migrations, then start the app with auto-reload
CMD ["/bin/sh", "-c", "/app/docker-entrypoint.sh"]