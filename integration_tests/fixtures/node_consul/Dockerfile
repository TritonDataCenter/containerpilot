FROM nginx:latest

RUN apt-get update && \
    apt-get install -y \
    curl \
    unzip && \
    rm -rf /var/lib/apt/lists/*

RUN curl -Lo /tmp/consul_template_0.11.0_linux_amd64.zip https://github.com/hashicorp/consul-template/releases/download/v0.11.0/consul_template_0.11.0_linux_amd64.zip && \
    unzip /tmp/consul_template_0.11.0_linux_amd64.zip && \
    mv consul-template /bin

ADD build/containerbuddy /bin/containerbuddy
ADD nginx.json /nginx.json

ADD reload-nginx.sh /reload-nginx.sh

ADD etc/nginx/conf.d /etc/nginx/conf.d/

ENTRYPOINT ["/bin/containerbuddy","-config=file:///nginx.json","nginx","-g","daemon off;"]
