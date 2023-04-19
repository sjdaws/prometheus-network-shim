FROM golang:1.20 AS builder
WORKDIR /go/src/github.com/sjdaws/prometheus-network-shim
COPY . .
RUN go build -ldflags "-X main.build=$(git rev-parse HEAD)" -o bin/prometheus-network-shim .
RUN chmod +x bin/prometheus-network-shim

FROM centos:7
LABEL io.k8s.display-name="prometheus-network-shim" \
    io.k8s.description="This is a shim for adding pod name to prometheus network metrics."
COPY --from=builder /go/src/github.com/sjdaws/prometheus-network-shim/bin/prometheus-network-shim /usr/bin/prometheus-network-shim
CMD ["/usr/bin/prometheus-network-shim"]
