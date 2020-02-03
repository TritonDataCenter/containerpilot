FROM golang:1.14

ENV CONSUL_VERSION=1.7.3

RUN  apt-get update \
     && apt-get install -y unzip \
     && go get golang.org/x/lint/golint

RUN export CONSUL_CHECKSUM=453814aa5d0c2bc1f8843b7985f2a101976433db3e6c0c81782a3c21dd3f9ac3 \
    && export archive=consul_${CONSUL_VERSION}_linux_amd64.zip \
    && curl -Lso /tmp/${archive} https://releases.hashicorp.com/consul/${CONSUL_VERSION}/${archive} \
    && echo "${CONSUL_CHECKSUM}  /tmp/${archive}" | sha256sum -c \
    && cd /bin \
    && unzip /tmp/${archive} \
    && chmod +x /bin/consul \
    && rm /tmp/${archive}

ENV CGO_ENABLED 0
ENV GOPATH /go:/cp
