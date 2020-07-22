ARG ALPINE_VERSION=3.10

FROM alpine:$ALPINE_VERSION

COPY build/cerberus-linux-amd64 /cerberusd
COPY static /static

ENTRYPOINT [ "/cerberusd" ]
