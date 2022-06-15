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

FROM frolvlad/alpine-bash:latest
RUN apk --no-cache add ca-certificates libstdc++
WORKDIR /root/
COPY --from=builder /go/src/github.com/vulcanize/ipld-eth-beacon-indexer/ipld-eth-beacon-indexer /root/ipld-eth-beacon-indexer
ADD entrypoint.sh .
ENTRYPOINT ["./entrypoint.sh"]