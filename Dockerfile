# Build Geth in a stock Go builder container
FROM golang:1.12-stretch as builder

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y gcc make git

ADD . /go-ethereum

WORKDIR /go-ethereum

RUN make cmd/geth

# Pull Geth into a second stage deploy ubuntu container
FROM ubuntu:18.04

RUN apt-get update && apt-get install -y openssh-server iputils-ping iperf3 && apt-get clean
COPY --from=builder /go-ethereum/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]