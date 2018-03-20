# Build regulatory in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . $GOPATH/src/github.com/maichain/eth-indexer
RUN cd $GOPATH/src/github.com/maichain/eth-indexer && make indexer && mv build/bin/indexer /indexer

# Pull regulatory into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /indexer /usr/local/bin/

ENTRYPOINT ["indexer"]
