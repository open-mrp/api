FROM alpine
WORKDIR /app

ADD build build

ENTRYPOINT build/billing-service
