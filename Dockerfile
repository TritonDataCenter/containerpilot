FROM golang:1.13

ENV CONSUL_VERSION=1.6.3

RUN  apt-get update \
     && apt-get install -y unzip \
     && go get golang.org/x/lint/golint \
     && curl --fail -Lso consul.zip "https://releases.hashicorp.com/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_amd64.zip" \
     && unzip consul.zip -d /usr/bin

ENV CGO_ENABLED 0
ENV GOPATH /go:/cp
