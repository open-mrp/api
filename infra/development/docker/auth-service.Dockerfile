FROM alpine
WORKDIR /app

ADD build build

ENTRYPOINT build/auth-service
