FROM alpine
WORKDIR /app

ADD shared shared
ADD build build
ADD services/auth-service/templates services/auth-service/templates


ENTRYPOINT build/auth-service
