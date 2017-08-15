FROM node:slim

RUN apt-get update && \
    apt-get install -y \
    curl netcat-openbsd && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g json http-server

COPY build/containerpilot /bin/containerpilot
COPY containerpilot.json5 /etc/containerpilot.json5
COPY containerpilot-with-coprocess.json5 /etc/containerpilot-with-coprocess.json5
COPY reload-app.sh /reload-app.sh
COPY reload-containerpilot.sh /reload-containerpilot.sh
COPY sensor.sh /sensor.sh

ENV CONTAINERPILOT=/etc/containerpilot.json5

# default port, allows us to override in docker-compose and also test
# env var interpolation in the command args
ENV APP_PORT=8000

ENTRYPOINT [ "/bin/containerpilot" ]
