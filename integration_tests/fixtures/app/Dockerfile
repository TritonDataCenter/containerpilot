FROM node:slim

RUN apt-get update && \
    apt-get install -y \
    curl && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g json http-server

COPY build/containerpilot /bin/containerpilot
COPY app-with-consul.json /app-with-consul.json
COPY app-with-coprocess.json /app-with-coprocess.json
COPY app-with-consul-prestart-sigusr1.json /app-with-consul-prestart-sigusr1.json
COPY app-with-consul-prestart-sighup.json /app-with-consul-prestart-sighup.json
COPY app-with-etcd.json /app-with-etcd.json
COPY reload-app.sh /reload-app.sh
COPY reload-app-etcd.sh /reload-app-etcd.sh
COPY reload-app-prestart.sh /reload-app-prestart.sh
COPY sensor.sh /sensor.sh
COPY task.sh /task.sh

# by default use app-consul.json, allows for override in docker-compose
ENV CONTAINERPILOT=file:///app-with-consul.json

# default port, allows us to override in docker-compose and also test
# env var interpolation in the command args
ENV APP_PORT=8000

ENTRYPOINT [ "/bin/containerpilot", \
             "/usr/local/bin/node", \
             "/usr/local/bin/http-server", "/srv", "-p", "{{ .APP_PORT }}"]
