# Nginx container including ContainerPilot
FROM alpine:3.3

# install nginx and tooling we need
RUN apk update && apk add \
    nginx \
    curl \
    unzip \
    && rm -rf /var/cache/apk/*

# we use consul-template to re-write our Nginx virtualhost config
RUN curl -Lo /tmp/consul_template_0.14.0_linux_amd64.zip https://releases.hashicorp.com/consul-template/0.14.0/consul-template_0.14.0_linux_amd64.zip && \
    unzip /tmp/consul_template_0.14.0_linux_amd64.zip && \
    mv consul-template /bin

# add ContainerPilot build and configuration
COPY build/containerpilot /bin/containerpilot
COPY etc /etc

EXPOSE 80

# by default use nginx-with-consul.json, allows for override in docker-compose
ENV CONTAINERPILOT=/etc/nginx-with-consul.json5

ENTRYPOINT [ "/bin/containerpilot" ]
