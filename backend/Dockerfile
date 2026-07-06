# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tesseract-ocr tesseract-ocr-data-hin wget

WORKDIR /app

COPY --from=builder /bin/api /bin/api
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/ready || exit 1

CMD ["/bin/api"]
