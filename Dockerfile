FROM golang

RUN  go get github.com/golang/lint/golint \
  && go get github.com/tools/godep

ENV CGO_ENABLED 0
ENV GO15VENDOREXPERIMENT 1
ENTRYPOINT ["make"]
