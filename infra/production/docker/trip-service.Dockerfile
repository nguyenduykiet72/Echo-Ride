FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/ride-service
RUN CGO_ENABLED=0 GOOS=linux go build -o ride-service ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/ride-service/trip-service .
CMD ["./trip-service"] 