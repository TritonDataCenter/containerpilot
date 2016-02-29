FROM golang:latest

RUN go get github.com/hashicorp/consul
RUN go get github.com/samalba/dockerclient
RUN go get github.com/coreos/etcd/client

COPY src /go/src/test_probe
RUN go install test_probe
