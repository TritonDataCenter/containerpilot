FROM golang:1.8

RUN  apt-get update \
     && apt-get install -y unzip \
     && go get github.com/golang/lint/golint \
     && curl -Lo /tmp/glide.tgz "https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz" \
     && tar -C /usr/bin -xzf /tmp/glide.tgz --strip=1 linux-amd64/glide \
     && curl --fail -Lso consul.zip              "https://releases.hashicorp.com/consul/0.7.5/consul_0.7.5_linux_amd64.zip" \
     && unzip consul.zip -d /usr/bin

ENV CGO_ENABLED 0
ENV GOPATH /go:/cp
