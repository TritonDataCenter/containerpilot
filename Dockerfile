FROM golang:1.18

ENV CONSUL_VERSION=1.11.4

RUN  apt-get update \
     && apt-get install -y unzip \
     && go install golang.org/x/lint/golint@latest


RUN export CONSUL_CHECKSUM=5155f6a3b7ff14d3671b0516f6b7310530b509a2b882b95b4fdf25f4219342c8 \
    && export archive=consul_${CONSUL_VERSION}_linux_amd64.zip \
    && curl -Lso /tmp/${archive} https://releases.hashicorp.com/consul/${CONSUL_VERSION}/${archive} \
    && echo "${CONSUL_CHECKSUM}  /tmp/${archive}" | sha256sum -c \
    && cd /bin \
    && unzip /tmp/${archive} \
    && chmod +x /bin/consul \
    && rm /tmp/${archive}

ENV CGO_ENABLED 0
ENV GOPATH /go
