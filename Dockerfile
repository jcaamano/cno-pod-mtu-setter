FROM registry.access.redhat.com/ubi8/go-toolset:latest AS builder
ENV WORKDIR /opt/app-root/src/go/src/github.com/jcaamano/cno-pod-mtu-setter
ENV GOBIN /usr/lib/golang/bin
USER root
WORKDIR $WORKDIR
COPY . $WORKDIR/
RUN go install ./...

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /usr/lib/golang/bin/cno-pod-mtu-setter /usr/local/bin
ENTRYPOINT ["/usr/local/bin/cno-pod-mtu-setter"]
