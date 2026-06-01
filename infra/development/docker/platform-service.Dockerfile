FROM alpine
WORKDIR /app

ADD build build

ENTRYPOINT build/platform-service
