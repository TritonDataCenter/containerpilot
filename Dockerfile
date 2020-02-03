FROM golang:1.13

ENV CONSUL_VERSION=1.6.3

RUN  apt-get update \
     && apt-get install -y unzip \
     && go get golang.org/x/lint/golint

RUN export CONSUL_CHECKSUM=3ada92a7b49c11076d0a2db9db4ad53ee366fcfb0e057118a322ad0daf188c60 \
    && export archive=consul_${CONSUL_VERSION}_linux_amd64.zip \
    && curl -Lso /tmp/${archive} https://releases.hashicorp.com/consul/${CONSUL_VERSION}/${archive} \
    && echo "${CONSUL_CHECKSUM}  /tmp/${archive}" | sha256sum -c \
    && cd /bin \
    && unzip /tmp/${archive} \
    && chmod +x /bin/consul \
    && rm /tmp/${archive}

ENV CGO_ENABLED 0
ENV GOPATH /go:/cp
