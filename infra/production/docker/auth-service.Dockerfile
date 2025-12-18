FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/auth-service/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/services/auth-service/cmd/auth-service .
COPY services/auth-service/templates ./templates
CMD ["./auth-service"]
