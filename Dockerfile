FROM golang:alpine AS builder
LABEL MAINTAINER="Faizan Bashir <faizan.ibn.bashir@gmail.com>"

WORKDIR /go/src/app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/todo

FROM alpine

# Labelling the image
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION

LABEL maintainer="faizanbashir" \
  org.opencontainers.image.created=$BUILD_DATE \
  org.opencontainers.image.url="https://github.com/faizanbashir/k8s-golang-mongodb" \
  org.opencontainers.image.source="https://github.com/faizanbashir/k8s-golang-mongodb" \
  org.opencontainers.image.version=$VERSION \
  org.opencontainers.image.revision=$REVISION \
  org.opencontainers.image.vendor="faizanbashir" \
  org.opencontainers.image.title="k8s-golang-mongodb" \
  org.opencontainers.image.description="Golang Todo API" \
  org.opencontainers.image.licenses="Apache"

RUN addgroup -S app \
  && adduser -S -g app app \
  && apk --no-cache add \
  curl openssl netcat-openbsd iputils

WORKDIR /home/app

COPY --from=builder /go/bin/todo .
RUN chown -R app:app ./

HEALTHCHECK CMD curl -sS http://localhost:8080/todo/healthz || exit 1

USER app

EXPOSE 8080

CMD [ "./todo" ]