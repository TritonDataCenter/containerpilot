FROM alpine:3.3

ADD build/containerbuddy /bin/containerbuddy
ADD app.json /app.json

ADD zombie.sh /zombie.sh
RUN chmod 755 /zombie.sh

ENTRYPOINT [\
  "/bin/containerbuddy","-config=file:///app.json",\
  "tail","-f"]
