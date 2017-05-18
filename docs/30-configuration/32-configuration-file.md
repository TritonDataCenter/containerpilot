# Configuration file

ContainerPilot expects a configuration file with details about what services it should register, how to check their health, and what to do at startup and shutdown, among others. There are two ways to specify the ContainerPilot configuration location:

1. An environment variable `CONTAINERPILOT`, pointing to the file location in the container.
2. As an argument to ContainerPilot via the `-config` flag.

##### Examples: specifying the configuration file path

```bash
# configure via passing a file argument
$ containerpilot -config /etc/containerpilot.json5

# configure via environment variable
$ export CONTAINERPILOT=/etc/containerpilot.json5
$ containerpilot
```

The configuration file format is [JSON5](http://json5.org/). If you are familiar with JSON, it is similar except that it accept comments, fields don't need to be surrounded by quotes, and it isn't nearly as fussy about extraneous trailing commas.

## Schema

The following is a completed example of the JSON5 file configuration schema, with all optional fields shown and fields annotated.

```json5
{
  consul: "localhost:8500",
  logging: {
    level: "INFO",
    format: "default",
    output: "stdout"
  },
  jobs: [
    {
      name: "app",
      exec: "/bin/app",
      restarts: "unlimited",
      port: 80,
      when: {
        // we want to start this job when the "setup" job has exited
        // with success but give up after 60 sec
        source: "setup",
        once: "exitSuccess",
        timeout: "60s"
      },
      health: {
        exec: "/usr/bin/curl --fail -s -o /dev/null http://localhost/app",
        interval: 5,
        tll: 10,
        timeout: "5s",
      },
      tags: [
        "app",
        "prod"
      ],
      interfaces: [
        "eth0",
        "eth1[1]",
        "192.168.0.0/16",
        "2001:db8::/64",
        "eth2:inet",
        "eth2:inet6",
        "inet",
        "inet6",
        "static:192.168.1.100", // a trailing comma isn't an error!
        ]
    },
    {
      name: "setup",
      // we can create a chain of "prestart" events
      when: {
        source: "consul-agent",
        once: "healthy"
      },
      exec: "/usr/local/bin/preStart-script.sh",
      restart: "never"
    },
    {
      name: "preStop",
      when: {
        source: "app",
        once: "stopping"
      },
      exec: "/usr/local/bin/preStop-script.sh",
      restart: "never",
    },
    {
      name: "postStop",
      when: {
        source: "app",
        once: "stopped"
      },
      exec: "/usr/local/bin/postStop-script.sh",
    },
    {
      // a service that doesn't have a "when" field starts up on the
      // global "startup" event by default
      name: "consul-agent",
      // note we don't have a port here because we don't intend to
      // advertise one to the service discovery backend
      exec: "consul -agent -join consul",
      restart: "always"
    },
    {
      name: "consul-template",
      exec: ["consul-template", "-consul", "consul",
             "-template", "/tmp/template.ctmpl:/tmp/result"],
      restart: "always",
    },
    {
      name: "periodic-task1",
      exec: "/usr/local/bin/task.sh arg1",
      timeout: "100ms",
      when: {
        interval: "1500ms"
      }
    },
    {
      name: "reload-app",
      when: {
        source: "watch.app",
        each: "changed"
      },
      exec: "/usr/local/bin/reload-app.sh",
      timeout: "10s"
    },
    {
      name: "reload-nginx",
      when: {
        source: "watch.nginx",
        each: "changed"
      },
      exec: "/usr/local/bin/reload-nginx.sh",
      timeout: "30s"
    }
  ],
  watches: {
    {
      name: "app",
      interval: 10
    },
    {
      name: "nginx",
      interval: 30
    }
  },
  control: {
    socket: "/var/run/containerpilot.socket"
  },
  telemetry: {
    port: 9090,
    interfaces: "eth0"
    sensors: [
      {
        name: "metric_id"
        help: "help text"
        type: "counter"
        interval: 5
        exec: "/usr/local/bin/sensor.sh"
      }
    ]
  }
}
```


### Consul

ContainerPilot uses Hashicorp's [Consul](https://www.consul.io/) to register jobs in the container as services. Watches look to Consul to find out the status of other services.

[Read more](./33-consul.md).

### Logging

The optional logging config adjusts the output format and verbosity of ContainerPilot logs. The default behavior is to log to `stdout` at `INFO` using the go [LstdFlags](https://golang.org/pkg/log/) format.

[Read more](./38-logging.md).

### Jobs

Jobs are the core user-defined concept in ContainerPilot. A job is a process and rules for when to execute it, how to health check it, and how to advertise it to Consul. The rules are intended to allow for flexibility to cover nearly any type of process one might want to run.

[Read more](./34-jobs.md).

### Watches

A watch is a configuration of a service to watch in Consul. The watch monitors the state of the service and emits events when the service becomes healthy, becomes unhealthy, or has a change in the number of instances. Note that a watch does not include a behavior; watches only emit the event so that jobs can consume that event.

[Read more](./35-watches.md).


### Control

Jobs often need a way to send information back to ContainerPilot to reload its own configuration, to update metrics, to put a service into maintenance mode, etc. ContainerPilot exposes a HTTP control plane that listens on a local unix socket. By default this can be found at `/var/run/containerpilot.socket`, and the location can be changed via the `control` configuration field.

[Read more](./37-control-plane.md).


### Telemetry

If a `telemetry` option is provided, ContainerPilot will expose a [Prometheus](http://prometheus.io) HTTP client interface that can be used to scrape performance telemetry. The telemetry interface is advertised as a service to the discovery service similar to services configured via the `services` block. Each `sensor` for the telemetry service will run periodically and record values in the [Prometheus client library](https://github.com/prometheus/client_golang). A Prometheus server can then make HTTP requests to the telemetry endpoint.

[Read more](./36-telemetry.md).


## Configuration extras

### Interfaces

The `interfaces` parameter allows for one or more specifications to be used when searching for the advertised IP. The first specification that matches stops the search process, so they should be ordered from most specific to least specific.

- `eth0` : Match the first IPv4 address on `eth0` (alias for `eth0:inet`)
- `eth0:inet6` : Match the first IPv6 address on `eth0`
- `eth0[1]` : Match the 2nd IP address on `eth0` (zero-based index)
- `10.0.0.0/16` : Match the first IP that is contained within the IP Network
- `fdc6:238c:c4bc::/48` : Match the first IP that is contained within the IPv6 Network
- `inet` : Match the first IPv4 Address (excluding `127.0.0.0/8`)
- `inet6` : Match the first IPv6 Address (excluding `::1/128`)
- `static:192.168.1.100` : Use this Address. Useful for all cases where the IP is not visible in the container

Interfaces and their IP addresses are ordered alphabetically by interface name, then by IP address (lexicographically by bytes).

**Sample ordering**

- `eth0 10.2.0.1 192.168.1.100`
- `eth1 10.0.0.100 10.0.0.200`
- `eth2 10.1.0.200 fdc6:238c:c4bc::1`
- `lo ::1 127.0.0.1`

## Exec and arguments

All `exec` fields, including `jobs/exec`, `jobs/health/exec`, and `telemetry/sensors/exec`, accept both a string or an array. If a string is given, the command and its arguments are separated by spaces; otherwise, the first element of the array is the command path, and the rest are its arguments. This is sometimes useful for breaking up long command lines.

**String command**

```json5
health: {
  exec: "/usr/bin/curl --fail -s http://localhost/app"
}
```

**Array command**

```json5
health: {
  exec: [
    "/usr/bin/curl",
    "--fail",
    "-s",
    "http://localhost/app"
  ]
}
```

## Environment variables

ContainerPilot will set the following environment variables for all its child processes. Note that these environment variables are not available during configuration [template parsing and rendering](#template-rendering), because they require that the template be rendered first.

- `CONTAINERPILOT_PID`: the PID of ContainerPilot itself.
- `CONTAINERPILOT_{JOB}_IP`: the IP address of every job that ContainerPilot advertises for service discovery.


## Template rendering

ContainerPilot configuration has template support. If you have an environment variable such as `FOO=BAR` then you can use `{{ .FOO }}` in your configuration file or in your command arguments and it will be substituted with `BAR`. The `CONTAINERPILOT_{JOB}_IP` environment variable that is set by the services configuration is available to child processes but not to the configuration file.

**Example usage in a config file**

```json5
{
  consul: consul:8500,
  job: {
   exec: "/bin/setup.sh {{.URL_TO_SERVICE}} {{.API_KEY}}",
  }
}
```

**Note**:  If you need more than just variable interpolation, check out the [Go text/template Docs](https://golang.org/pkg/text/template/). ContainerPilot ships with the template functions from the stdlib, as well as some extensions:

##### `default`

Pprovides a default value if the variable is empty. For example: `{{ .CONSUL | default "localhost}}` would output `localhost` if the `CONSUL` env var is not set.

##### `split` and `join`

Split a string into parts, or join them together. For example, if we have the environment variable `PARTS=a:b:c`, then the following template: `Hello, {{.PARTS | split ":" | join "." }}!` would result in the output `Hello, a.b.c!`


##### `replaceAll` and `regexReplaceAll`

Replace a substring with another string (possibly using a regex for substring selection). For example, assume we have the environment variable `NAME=Template`:

- `Hello, {{.NAME | replaceAll "e" "_" }}!` will output `Hello, T_mplat_!`
- `Hello, {{.NAME | regexReplaceAll "[epa]+" "_" }}!` will output `Hello, T_m_l_t_!`
