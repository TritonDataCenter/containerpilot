FROM alpine:3.3

COPY build/containerpilot /bin/containerpilot
COPY app.json5 /app.json5
COPY zombie.sh /zombie.sh

ENTRYPOINT ["/bin/containerpilot", "-config=/app.json5"]
