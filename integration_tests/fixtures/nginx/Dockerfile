# Nginx container including Containerbuddy
FROM alpine:3.3

# install nginx and tooling we need
RUN apk update && apk add \
    nginx \
    curl \
    unzip \
    && rm -rf /var/cache/apk/*

# we use consul-template to re-write our Nginx virtualhost config
RUN curl -Lo /tmp/consul_template_0.11.0_linux_amd64.zip https://github.com/hashicorp/consul-template/releases/download/v0.11.0/consul_template_0.11.0_linux_amd64.zip && \
    unzip /tmp/consul_template_0.11.0_linux_amd64.zip && \
    mv consul-template /bin

# Add confd to re-write our Nginx config from etcd
RUN curl -Lo /bin/confd https://github.com/kelseyhightower/confd/releases/download/v0.11.0/confd-0.11.0-linux-amd64 && \
    chmod 755 /bin/confd

# add Containerbuddy build and configuration
COPY build/containerbuddy /bin/containerbuddy
COPY etc /etc

EXPOSE 80

# by default use nginx-with-consul.json, allows for override in docker-compose
ENV CONTAINERBUDDY=file:///etc/nginx-with-consul.json

ENTRYPOINT [ "/bin/containerbuddy", \
             "nginx", "-g", "daemon off;"]
