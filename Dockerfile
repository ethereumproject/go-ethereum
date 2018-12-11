# Build Geth in a stock Go builder container
FROM golang:1.11-alpine as builder

RUN apk add --no-cache bash git make gcc musl-dev linux-headers

RUN go get -v github.com/ethereumproject/go-ethereum/...
RUN go install github.com/ethereumproject/go-ethereum/cmd/geth
RUN cp -R ./bin /usr/local/bin/

EXPOSE 8545:8546 30303:30303/udp
ENTRYPOINT ["geth"]

