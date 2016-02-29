FROM golang:latest

RUN go get github.com/hashicorp/consul
RUN go get github.com/samalba/dockerclient

COPY src /go/src/test_probe
RUN go install test_probe
