FROM node:slim

RUN apt-get update && \
    apt-get install -y \
    curl && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g json http-server

COPY build/containerbuddy /bin/containerbuddy
COPY app-with-consul.json /app-with-consul.json
COPY app-with-etcd.json /app-with-etcd.json
COPY reload-app.sh /reload-app.sh
COPY reload-app-etcd.sh /reload-app-etcd.sh
COPY sensor.sh /sensor.sh

# by default use app-consul.json, allows for override in docker-compose
ENV CONTAINERBUDDY=file:///app-with-consul.json

ENTRYPOINT [ "/bin/containerbuddy", \
             "/usr/local/bin/node", \
             "/usr/local/bin/http-server", "/srv", "-p", "8000"]
