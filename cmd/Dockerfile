# Build indexer in a stock Go builder container
FROM golang:1.10.3-alpine3.8 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . $GOPATH/src/github.com/getamis/eth-indexer
RUN mkdir /indexer
RUN cd $GOPATH/src/github.com/getamis/eth-indexer && make all && mv build/bin/* /indexer

# Pull indexer into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /indexer/* /usr/local/bin/

RUN addgroup -S app -g 699
RUN adduser -S -u 699 -G app -s /sbin/nologin app

USER app
