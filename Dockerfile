# Use official Go image
FROM golang:1.25-alpine

# Install required tools
RUN apk add --no-cache bash git

# Install Air with a Go version compatible release
RUN go install github.com/air-verse/air@v1.61.1

# Add Go bin to PATH
ENV PATH=$PATH:/go/bin

# Set working directory
WORKDIR /app

# Copy go.mod & go.sum first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project
COPY . .

# Expose API port
EXPOSE 8080

# Run the app with auto-reload
CMD ["air", "-c", ".air.toml"]
