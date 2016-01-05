# a minimal Node.js container including containerbuddy
FROM node:slim

# install curl
RUN apt-get update && \
    apt-get install -y \
    curl && \
    rm -rf /var/lib/apt/lists/*

# install a simple http server
RUN npm install -g json http-server

# add containerbuddy and all our configuration
ADD opt/containerbuddy /opt/containerbuddy/
