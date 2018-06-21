FROM golang:1.10 as builder

ARG CC=""
ARG CC_PKG=""
ARG CC_GOARCH=""

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    if [ -n "$CC_PKG" ]; then \
      apt-get install -y $CC_PKG; \
    fi

ADD . /go/src/github.com/rycus86/github-prometheus-exporter
WORKDIR /go/src/github.com/rycus86/github-prometheus-exporter

RUN export CC=$CC && \
    export GOOS=linux && \
    export GOARCH=$CC_GOARCH && \
    export CGO_ENABLED=0 && \
    go build -o /var/tmp/app -v .

FROM scratch

LABEL maintainer "Viktor Adam <rycus86@gmail.com>"

COPY --from=builder /var/tmp/app /exporter
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT [ "/exporter" ]
