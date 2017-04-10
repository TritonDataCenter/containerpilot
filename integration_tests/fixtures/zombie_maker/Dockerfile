FROM alpine:3.3

ADD build/containerpilot /bin/containerpilot
ADD app.json5 /app.json5

ADD zombie.sh /zombie.sh
RUN chmod 755 /zombie.sh

ENTRYPOINT ["/bin/containerpilot", "-config=file:///app.json5"]
