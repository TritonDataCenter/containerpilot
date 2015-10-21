# a minimal Nginx container including containerbuddy and a simple virtulhost config
FROM nginx:latest

# install curl and jq
RUN apt-get update && \
    apt-get install -y \
    curl \
    unzip && \
    rm -rf /var/lib/apt/lists/*

RUN curl -Lo /tmp/consul_template_0.11.0_linux_amd64.zip https://github.com/hashicorp/consul-template/releases/download/v0.11.0/consul_template_0.11.0_linux_amd64.zip && \
    unzip /tmp/consul_template_0.11.0_linux_amd64.zip && \
    mv consul-template /bin

# add containerbuddy and all our configuration
ADD opt/containerbuddy /opt/containerbuddy/
ADD etc/nginx/conf.d /etc/nginx/conf.d/
