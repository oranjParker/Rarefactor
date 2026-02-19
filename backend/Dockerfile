# --- Build Stage ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/worker/main.go

# --- Run Stage ---
FROM alpine:latest

RUN apk add --no-cache \
         ca-certificates \
         chromium \
         harfbuzz \
         freetype \
         ttf-freefont \
         nss
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 50051 8000
CMD ["./main"]