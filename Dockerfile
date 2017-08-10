FROM golang:1.8-alpine

MAINTAINER Sebastian Montini <sebastian@jampp.com>

RUN apk update \
  && apk add gcc musl-dev git linux-headers make

RUN go get github.com/pshima/consul-snapshot

COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

ENTRYPOINT [ "/docker-entrypoint.sh" ]
