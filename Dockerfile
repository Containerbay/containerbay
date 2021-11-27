FROM golang as builder
RUN apt-get update && apt-get install -y upx
ADD . /containerbay
RUN cd /containerbay && CGO_ENABLED=0 go build

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /containerbay/containerbay /usr/bin/containerbay

ENTRYPOINT ["/usr/bin/containerbay"]