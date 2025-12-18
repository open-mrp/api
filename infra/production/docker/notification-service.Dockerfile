FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/notification-service/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/notification-service/cmd/notification-service .
CMD ["./notification-service"]