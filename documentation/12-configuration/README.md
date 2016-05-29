Title: Configuration
----
Text:

ContainerPilot expects a configuration file with details about what services it should register, how to check their health, and what to do at startup and shutdown, among others.

### Specifying the configuration file

There are a few ways to specify the ContainerPilot configuration location.

1. An environment variable set in the `docker run...` string, in a Docker Compose manifest, in the Dockerfile, or elsewhere
2. As an argument to ContainerPilot, perhaps in the the `docker run...` string, or in in the `CMD` or `ENTRYPOINT` of the Dockerfile
3. As a JSON string passed to ContainerPilot in any of the locations named above

#### Examples: specifying the configuration file path

```bash
# configure via passing a file argument
$ containerpilot -config file:///etc/containerpilot.json myapp --args --for --my --app

# configure via environment variable
$ export CONTAINERPILOT=file:///etc/containerpilot.json
$ containerpilot myapp --args --for --my --app
```

Many of the Autopilot Pattern implementations specify the configuration file path via an environment variable in the Dockerfile. [See the Nginx implementation for an example](https://github.com/autopilotpattern/nginx/blob/master/Dockerfile).

### Configuration file

The format of the JSON file configuration is as follows:

```json
{
  "consul": "consul:8500",
  "preStart": "/usr/local/bin/preStart-script.sh {{.ENV_VAR_NAME}}",
  "logging": {
    "level": "INFO",
    "format": "default",
    "output": "stdout"
  },
  "stopTimeout": 5,
  "preStop": "/usr/local/bin/preStop-script.sh",
  "postStop": "/usr/local/bin/postStop-script.sh",
  "services": [
    {
      "name": "app",
      "port": 80,
      "health": [
        "/usr/bin/curl",
        "--fail",
        "-s",
        "http://localhost/app"
      ],
      "interfaces": [
        "eth0",
        "eth1[1]",
        "192.168.0.0/16",
        "2001:db8::/64",
        "eth2:inet",
        "eth2:inet6",
        "inet",
        "inet6"
      ],
      "poll": 10,
      "ttl": 30,
      "tags": ["tag1"]
    }
  ],
  "backends": [
    {
      "name": "nginx",
      "poll": 30,
      "onChange": "/usr/local/bin/reload-app.sh"
    },
    {
      "name": "app",
      "poll": 10,
      "onChange": "/usr/local/bin/reload-app.sh"
    }
  ],
  "telemetry": {
    "port": 9090,
    "sensors": [
       {
        "name": "metric_id",
        "help": "help text",
        "type": "counter",
        "poll": 5,
        "check": ["/usr/local/bin/sensor.sh"]
      }
    ]
  },
  "tasks": [
    {
      "name": "task",
      "command": ["/usr/local/bin/task.sh","arg1"],
      "frequency": "1500ms",
      "timeout": "100ms"
    }
  ]
}
```

### `services`

- `name` is the name of the service as it will appear in Consul. Each instance of the service will have a unique ID made up from `name`+hostname of the container.
- `port` is the port the service will advertise to Consul.
- `health` is the executable (and its arguments) used to check the health of the service.
- `interfaces` is an optional single or array of interface specifications. If given, the IP of the service will be obtained from the first interface specification that matches. (Default value is `["eth0:inet"]`)
- `poll` is the time in seconds between polling for health checks.
- `ttl` is the time-to-live of a successful health check. This should be longer than the polling rate so that the polling process and the TTL aren't racing; otherwise Consul will mark the service as unhealthy.
- `tags` is an optional array of tags. If the discovery service supports it (Consul does), the service will register itself with these tags.

### `backends`

- `name` is the name of a backend service that this container depends on, as it will appear in Consul.
- `poll` is the time in seconds between polling for changes.
- `onChange` is the executable (and its arguments) that is called when there is a change in the list of IPs and ports for this backend.

### Service catalog

The service catalog (Consul and Etcd are both supported, others can be added) is where ContainerPilot registers the service(s) in the container, and where it looks to see what other services are registered. ContainerPilot works in conjunction with the service catalog of your choice as a complete service discovery solution.

- `consul` configures discovery via [Hashicorp Consul](https://www.consul.io/). For use with Consul's [ACL system](https://www.consul.io/docs/internals/acl.html), use the `CONSUL_HTTP_TOKEN` environment variable. Expects `hostname:port` string. If you are communicating with Consul over TLS you may include the scheme (ex. `https://consul:8500`):

    ```
    "consul": "consul:8500"
    ```

- `etcd` configures discovery via [CoreOS etcd](https://coreos.com/etcd/). Expects a config object:

    ```
    "etcd": {
        "endpoints": [
            "http://etcd:4001"
        ],
        "prefix": "/containerpilot"
    }
    ```

    - `endpoints` is the list of etcd nodes in your cluster
    - `prefix` is the path that will be prefixed to all service discovery keys. This key is optional. (Default: `/containerpilot`)

### `logging`

The optional logging config adjusts the output format and verbosity of ContainerPilot logs.

- `level` adjusts the verbosity of the messages output by containerpilot. Must be one of: `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`, `PANIC` (Default is `INFO`)
- `format` adjust the output format for log messages. Can be `default`, `text`, or `json` (Default is `default`)
- `output` picks the output stream for log messages. Can be `stderr` or `stdout` (Default is `stdout`)

Processes which are run by ContainerPilot, such as `health`, lifecycle hooks (`preStart`,`preStop`,`postStop`,`onChange`), `task` and `sensor` output are captured and streamed to the logging framework. `stdout` creates `INFO` logs, and `stderr` creates `DEBUG` logs.

This configuration does not affect the output of the shimmed application, which outputs directly to `stdout` and `stderr`.

Logging Format Examples:

`default` - Go log package with [LstdFlags](https://golang.org/pkg/log/)

```
2015/03/26 01:27:38 Started observing beach
2015/03/26 01:27:38 A group of walrus emerges from the ocean
2015/03/26 01:27:38 The group's number increased tremendously!
2015/03/26 01:27:38 Temperature changes
2015/03/26 01:27:38 It's over 9000!
2015/03/26 01:27:38 The ice breaks!
```

`text` - [logrus TextFormatter](https://github.com/Sirupsen/logrus)

```
time="2015-03-26T01:27:38-04:00" level=debug msg="Started observing beach" animal=walrus number=8
time="2015-03-26T01:27:38-04:00" level=info msg="A group of walrus emerges from the ocean" animal=walrus size=10
time="2015-03-26T01:27:38-04:00" level=warning msg="The group's number increased tremendously!" number=122 omg=true
time="2015-03-26T01:27:38-04:00" level=debug msg="Temperature changes" temperature=-4
time="2015-03-26T01:27:38-04:00" level=panic msg="It's over 9000!" animal=orca size=9009
time="2015-03-26T01:27:38-04:00" level=fatal msg="The ice breaks!" err=&{0x2082280c0 map[animal:orca size:9009] 2015-03-26 01:27:38.441574009 -0400 EDT panic It's over 9000!} number=100 omg=true
exit status 1
```

`json` - [logrus JSONFormatter](https://github.com/Sirupsen/logrus)

```
{"animal":"walrus","level":"info","msg":"A group of walrus emerges from the ocean","size":10,"time":"2014-03-10 19:57:38.562264131 -0400 EDT"}
{"level":"warning","msg":"The group's number increased tremendously!","number":122,"omg":true,"time":"2014-03-10 19:57:38.562471297 -0400 EDT"}
{"animal":"walrus","level":"info","msg":"A giant walrus appears!","size":10,"time":"2014-03-10 19:57:38.562500591 -0400 EDT"}
{"animal":"walrus","level":"info","msg":"Tremendously sized cow enters the ocean.","size":9,"time":"2014-03-10 19:57:38.562527896 -0400 EDT"}
{"level":"fatal","msg":"The ice breaks!","number":100,"omg":true,"time":"2014-03-10 19:57:38.562543128 -0400 EDT"}
```

Logging details here do not affect how the Docker daemon (or other container runtime) handles logging. [See this blog post for a narrative and examples of how to manage log output from the container](https://www.joyent.com/blog/docker-log-drivers).

### `telemetry`

If a `telemetry` option is provided, ContainerPilot will expose a [Prometheus](http://prometheus.io) HTTP client interface that can be used to scrape performance telemetry. The telemetry interface is advertised as a service to the discovery service similar to services configured via the `services` block. Each `sensor` for the telemetry service will run periodically and record values in the [Prometheus client library](https://github.com/prometheus/client_golang). A Prometheus server can then make HTTP requests to the telemetry endpoint.

[Read more](/containerpilot/docs/telemetry).

### `tasks`

Tasks are commands that are run periodically. They are typically used to perform housekeeping such as incremental back-ups, or pushing metrics to systems that cannot collect metrics through service discovery like Prometheus.

[Read more](/containerpilot/docs/tasks).

### Lifecycle fields

- `preStart`, `preStop`, `postStop` represent specific [events in the application's lifecycle](/containerpilot/docs/lifecycle), and [have their own section in the docs](/containerpilot/docs/start-stop).
- `stopTimeout` Optional amount of time in seconds to wait before killing the application. (defaults to `5`). Providing `-1` will kill the application immediately.

### `interfaces`

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

### Commands & arguments

All executable fields, including `services/health`, `preStart`, `preStop`, `postStop`, `backends/onChange`, `task/command`, and `telemetry/sensors/check`, accept both a string or an array. If a string is given, the command and its arguments are separated by spaces; otherwise, the first element of the array is the command path, and the rest are its arguments.

**String command**

```json
"health": "/usr/bin/curl --fail -s http://localhost/app"
```

**Array command**

```json
"health": [
  "/usr/bin/curl",
  "--fail",
  "-s",
  "http://localhost/app"
]
```

### Template configuration

ContainerPilot configuration has template support. If you have an environment variable such as `FOO=BAR` then you can use `{{.FOO}}` in your configuration file and it will be substituted with `BAR`.

**Example usage**

```json
{
  "consul": "consul:8500",
  "preStart": "/usr/local/bin/preStart-script.sh {{.URL_TO_SERVICE}} {{.API_KEY}}",
}
```

**Note**:  If you need more than just variable interpolation, check out the [Go text/template Docs](https://golang.org/pkg/text/template/).
