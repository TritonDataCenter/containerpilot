FROM node:slim

RUN apt-get update && \
    apt-get install -y \
    curl \
    unzip && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g json http-server

ADD build/containerbuddy /bin/containerbuddy
ADD app.json /app.json

ADD reload-app.sh /reload-app.sh
RUN chmod 755 /reload-app.sh

ENTRYPOINT [\
  "/bin/containerbuddy","-config=file:///app.json",\
  "/usr/local/bin/node","/usr/local/bin/http-server","/srv","-p","8000"]
