# Build stage
FROM golang:1.22.1@sha256:46a86b411554728154e56f9719426a47e5ded3cab7adb1ecb22a988f486e99ae AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o vresq -ldflags="-s -w" ./main.go

# Final image
FROM golang:1.22.1@sha256:46a86b411554728154e56f9719426a47e5ded3cab7adb1ecb22a988f486e99ae
COPY --from=builder /app/vresq /usr/local/bin/vresq