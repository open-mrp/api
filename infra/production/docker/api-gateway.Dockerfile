FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/api-gateway/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/api-gateway/cmd/api-gateway .
CMD ["./api-gateway"]