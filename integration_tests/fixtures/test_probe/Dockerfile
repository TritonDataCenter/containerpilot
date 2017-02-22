FROM golang:1.8

RUN go get github.com/tools/godep

COPY src /go/src/test_probe
RUN cd /go/src/test_probe && godep restore
RUN go install test_probe
