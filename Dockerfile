FROM golang:1.8.3 as builder
WORKDIR /go/src/github.com/willis7/status
COPY ./ ./
RUN CGO_ENABLED=0 go install -a -tags status -ldflags '-extldflags "-static"'
RUN ldd /go/bin/status | grep -q "not a dynamic executable"

FROM alpine
RUN  apk add --update --no-cache netcat-openbsd \
    curl
COPY --from=builder /go/bin/status /status
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs
COPY config.json config.json

EXPOSE 3000

CMD ["/status", "config.json"]
