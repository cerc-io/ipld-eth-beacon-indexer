FROM golang:1.18-alpine as builder

WORKDIR /go/src/github.com/vulcanize/ipld-eth-beacon-indexer
RUN apk --no-cache add ca-certificates make git g++ linux-headers libstdc++

ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod tidy; go mod download
COPY . .

RUN GCO_ENABLED=0 GOOS=linux go build -race -ldflags="-s -w" -o ipld-eth-beacon-indexer .
RUN chmod +x ipld-eth-beacon-indexer

FROM alpine:latest
RUN apk --no-cache add ca-certificates libstdc++ busybox-extras gettext libintl bash gawk sed grep bc coreutils
WORKDIR /root/
COPY --from=builder /go/src/github.com/vulcanize/ipld-eth-beacon-indexer/ipld-eth-beacon-indexer /root/ipld-eth-beacon-indexer
ADD entrypoint.sh .
ADD ipld-eth-beacon-config-docker.json .
ENTRYPOINT ["./entrypoint.sh"]
