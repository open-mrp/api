FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/logging-service/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o logging-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/logging-service/cmd/logging-service .
CMD ["./logging-service"]