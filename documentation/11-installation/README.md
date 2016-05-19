Title: Installation in a container
----
Text:

The best way to install ContainerPilot in a Docker container is by including it in the Dockerfile:

```
# get ContainerPilot release
ENV CONTAINERPILOT_VERSION 2.1.2
RUN export CP_SHA1=c31333047d58ba09d647d717ae1fc691133db6eb \
    && curl -Lso /tmp/containerpilot.tar.gz \
         "https://github.com/joyent/containerpilot/releases/download/${CONTAINERPILOT_VERSION}/containerpilot-${CONTAINERPILOT_VERSION}.tar.gz" \
    && echo "${CP_SHA1}  /tmp/containerpilot.tar.gz" | sha1sum -c \
    && tar zxf /tmp/containerpilot.tar.gz -C /bin \
    && rm /tmp/containerpilot.tar.gz
```

The above snippet adds ContainerPilot to the container. It also specifies the version to install and validates the application fingerprint to make sure that it's installing exactly the version you want.

The latest [ContainerPilot releases are available in Github](https://github.com/joyent/containerpilot/releases).

ContainerPilot wraps your primary application so it can pass signals and receive its exit code. To make this work, simply prefix the command or entrypoint for your application with ContainerPilot, similar to the following:

```
CMD [ "/usr/local/bin/containerpilot", \
    "nginx", \
        "-g", \
        "daemon off;"]
```

### In context

The following Dockerfile comes from [the Autopilot Pattern implementation of Nginx](https://github.com/autopilotpattern/nginx/blob/master/Dockerfile):

```
# A minimal Nginx container including ContainerPilot and a simple virtualhost config
FROM nginx:latest

# Add some stuff via apt-get
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bc \
        curl \
        unzip \
    && rm -rf /var/lib/apt/lists/*

# Add Consul template
# Releases at https://releases.hashicorp.com/consul-template/
ENV CONSUL_TEMPLATE_VERSION 0.14.0
ENV CONSUL_TEMPLATE_SHA1 7c70ea5f230a70c809333e75fdcff2f6f1e838f29cfb872e1420a63cdf7f3a78

RUN curl --retry 7 -Lso /tmp/consul-template.zip "https://releases.hashicorp.com/consul-template/${CONSUL_TEMPLATE_VERSION}/consul-template_${CONSUL_TEMPLATE_VERSION}_linux_amd64.zip" \
    && echo "${CONSUL_TEMPLATE_SHA1}  /tmp/consul-template.zip" | sha256sum -c \
    && unzip /tmp/consul-template.zip -d /usr/local/bin \
    && rm /tmp/consul-template.zip

# Add Containerpilot and set its configuration
ENV CONTAINERPILOT_VER 2.1.0
ENV CONTAINERPILOT file:///etc/containerpilot.json

RUN export CONTAINERPILOT_CHECKSUM=e7973bf036690b520b450c3a3e121fc7cd26f1a2 \
    && curl -Lso /tmp/containerpilot.tar.gz \
         "https://github.com/joyent/containerpilot/releases/download/${CONTAINERPILOT_VER}/containerpilot-${CONTAINERPILOT_VER}.tar.gz" \
    && echo "${CONTAINERPILOT_CHECKSUM}  /tmp/containerpilot.tar.gz" | sha1sum -c \
    && tar zxf /tmp/containerpilot.tar.gz -C /usr/local/bin \
    && rm /tmp/containerpilot.tar.gz

# Add our configuration files and scripts
COPY etc /etc
COPY bin /usr/local/bin

CMD [ "/usr/local/bin/containerpilot", \
    "nginx", \
        "-g", \
        "daemon off;"]
```