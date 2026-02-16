# Build Stage
FROM golang:alpine AS builder
WORKDIR /app

# Network & SSL Fixes
RUN apk update && apk add --no-cache git ca-certificates
RUN git config --global http.sslVerify false
ENV GOPROXY=direct
ENV GOINSECURE=*
ENV GONOSUMDB=*

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# 1. Install Swag (The Fix)
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy Source Code
COPY . .

# 2. Generate Docs (The Fix)
# This creates the 'docs' folder that main.go is looking for
RUN swag init -g cmd/api/main.go --parseDependency --parseInternal

# Build
RUN go build -o fraud-engine cmd/api/main.go

# Run Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/fraud-engine .
# Optional: Copy docs if you want to inspect them in the container
COPY --from=builder /app/docs ./docs 
CMD ["./fraud-engine"]