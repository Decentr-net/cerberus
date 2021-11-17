ARG ALPINE_VERSION=3.10
ARG GOLANG_VERSION=1.15.5

FROM golang:${GOLANG_VERSION}
WORKDIR /go/src/github.com/Decentr-net/cerberus/
COPY . .
RUN make linux

FROM alpine:${ALPINE_VERSION}
RUN apk update && apk add ca-certificates
COPY --from=0 /go/src/github.com/Decentr-net/cerberus/build/cerberus-linux-amd64 /cerberusd
COPY --from=0 /go/src/github.com/Decentr-net/cerberus/build/processor-linux-amd64 /processord
COPY --from=0 /go/src/github.com/Decentr-net/cerberus/build/sync-linux-amd64 /syncd
COPY static /static
COPY configs/rewards.yml /configs/rewards.yml
COPY scripts/migrations /migrations
ENTRYPOINT [ "/cerberusd" ]
