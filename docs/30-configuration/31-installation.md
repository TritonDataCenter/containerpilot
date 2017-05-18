# Installation

ContainerPilot is a statically-linked binary. Pre-compiled releases for Linux can be found on the [project releases page on GitHub](https://github.com/joyent/containerpilot/releases). The best way to install ContainerPilot in a Docker container is by including it in the Dockerfile:

```
# get ContainerPilot release
ENV CONTAINERPILOT_VERSION 3.0.0
RUN export checksum=a669f76aae9b3472f01eaf63f13e824fc2434692 \
    && curl -Lso /tmp/containerpilot.tar.gz \
         "https://github.com/joyent/containerpilot/releases/download/${CONTAINERPILOT_VERSION}/containerpilot-${CONTAINERPILOT_VERSION}.tar.gz" \
    && echo "${checksum}  /tmp/containerpilot.tar.gz" | sha1sum -c \
    && tar zxf /tmp/containerpilot.tar.gz -C /usr/local/bin \
    && rm /tmp/containerpilot.tar.gz

# add config file
COPY containerpilot.json5 /etc/containerpilot.json5

CMD /usr/local/bin/containerpilot -config /etc/containerpilot.json5
```

The above snippet adds the ContainerPilot binary to the container at `/usr/local/bin/containerpilot`. It also specifies the version to install and validates the application fingerprint to make sure that it's installing exactly the version you want.


### Building yourself

ContainerPilot is written in [go](https://golang.org/). The makefile at the root of the repository can build either via your local golang toolchain (currently golang 1.8) or in a Docker container.

If you have Docker running, `make build` will build a container image that includes the golang toolchain, downloads and installs all the required libraries into the `vendor/` directory, and builds ContainerPilot. The compiled binary will be found at `build/containerpilot`

If you have a golang toolchain, `make local build` will build using the architecture flags it picks up from the environment. This has mostly been tested on MacOS and SmartOS. It's unlikely that ContainerPilot will operate correctly when built for Windows because of specific POSIX behaviors it needs as an init system.
