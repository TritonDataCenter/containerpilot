FROM golang:1.19

ENV CONSUL_VERSION=1.13.3

RUN  apt-get update \
     && apt-get install -y unzip \
     && go install honnef.co/go/tools/cmd/staticcheck@latest

RUN export CONSUL_CHECKSUM=5370b0b5bf765530e28cb80f90dcb47bd7d6ba78176c1ab2430f56e460ed279c \
    && export archive=consul_${CONSUL_VERSION}_linux_amd64.zip \
    && curl -Lso /tmp/${archive} https://releases.hashicorp.com/consul/${CONSUL_VERSION}/${archive} \
    && echo "${CONSUL_CHECKSUM}  /tmp/${archive}" | sha256sum -c \
    && cd /bin \
    && unzip /tmp/${archive} \
    && chmod +x /bin/consul \
    && rm /tmp/${archive}

ENV CGO_ENABLED 0
