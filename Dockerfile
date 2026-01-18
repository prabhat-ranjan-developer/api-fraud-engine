# Build Stage
FROM golang:alpine AS builder
WORKDIR /app

# Network & SSL Fixes
RUN apk update && apk add --no-cache git ca-certificates
RUN git config --global http.sslVerify false
ENV GOPROXY=direct
ENV GOINSECURE=*
ENV GONOSUMDB=*

# Depdendencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN go build -o fraud-engine cmd/api/main.go

# Run Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/fraud-engine .
CMD ["./fraud-engine"]