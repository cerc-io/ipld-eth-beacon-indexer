FROM golang:1.18-alpine as builder

WORKDIR /go/src/github.com/vulcanize/ipld-ethcl-indexer
RUN apk --no-cache add ca-certificates make git g++ linux-headers

ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod tidy; go mod download
COPY . .

RUN GCO_ENABLED=0 GOOS=linux go build -race -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ipld-ethcl-indexer .
RUN chmod +x ipld-ethcl-indexer

FROM frolvlad/alpine-bash:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/vulcanize/ipld-ethcl-indexer/ipld-ethcl-indexer /root/ipld-ethcl-indexer
ADD entrypoint.sh .
ENTRYPOINT ["./entrypoint.sh"]