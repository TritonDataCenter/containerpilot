FROM golang:latest

ADD src /go/src/test_probe
RUN go get github.com/hashicorp/consul
RUN go get github.com/samalba/dockerclient

RUN go install test_probe
