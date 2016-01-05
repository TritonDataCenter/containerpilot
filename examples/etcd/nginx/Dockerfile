# a minimal Nginx container including containerbuddy and a simple virtulhost config
FROM nginx:latest

# install curl and jq
RUN apt-get update && \
    apt-get install -y \
    curl \
    unzip && \
    rm -rf /var/lib/apt/lists/*

# Add confd for templateing
RUN curl -Lo /bin/confd https://github.com/kelseyhightower/confd/releases/download/v0.11.0/confd-0.11.0-linux-amd64 && \
    chmod 755 /bin/confd

# add containerbuddy and all our configuration
ADD opt/containerbuddy /opt/containerbuddy/
ADD etc/nginx/conf.d /etc/nginx/conf.d/
ADD etc/confd /etc/confd
