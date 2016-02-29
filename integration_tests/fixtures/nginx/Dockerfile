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

# add Containerbuddy build and configuration
COPY build/containerbuddy /bin/containerbuddy
COPY nginx.json /etc/containerbuddy/nginx.json
COPY nginx.ctmpl /etc/containerbuddy/nginx.ctmpl

EXPOSE 80

ENTRYPOINT [ "/bin/containerbuddy", \
             "-config=file:///etc/containerbuddy/nginx.json", \
             "nginx", "-g", "daemon off;"]
