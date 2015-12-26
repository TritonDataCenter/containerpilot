FROM golang

RUN go get -u github.com/golang/lint/golint

ENV CGO_ENABLED 0
ENTRYPOINT ["make"]
