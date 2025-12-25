# Use official Go image
FROM golang:1.25-alpine

# Install required tools
RUN apk add --no-cache bash git

# Install Air (hot reload tool)
RUN go install github.com/air-verse/air@latest

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

# Run Air with config
CMD ["air", "-c", ".air.toml"]
