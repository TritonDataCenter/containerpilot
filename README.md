# ContainerPilot

*A service for assisting discovery and configuration of applications running in containers.*

[![Build Status](https://travis-ci.org/joyent/containerpilot.svg)](https://travis-ci.org/joyent/containerpilot)
[![MPL licensed](https://img.shields.io/badge/license-MPL_2.0-blue.svg)](https://github.com/joyent/containerpilot/blob/master/LICENSE)

### News: Containerbuddy is now ContainerPilot

We've renamed Containerbuddy to ContainerPilot to simplify and clarify the relationship between [autopilot pattern](http://autopilotpattern.io)—the approach to automating our applications—and [ContainerPilot](https://www.joyent.com/containerpilot)—the shared library that makes it easy to build autopilot pattern applications. [Please see the Joyent blog for more details](https://www.joyent.com/blog/containerbuddy-is-now-containerpilot).

### Container-native applications vs all the rest

Applications in containers typically need to talk to a source of truth to discover their upstream services and tell their downstream services where to find them. Container-native applications come into the world understanding this responsibility, but no one wants to rewrite all our current applications to do this.

We can wrap each application in a shell script that registers itself with the discovery service easily enough, but watching for changes to that service and ensuring that health checks are being made is more complicated. We can put a second process in the container, but unless we make a supervisor as PID1 then there's no way of knowing whether our shimmed process has died.

Additionally, discovery services like Consul provide a means of performing health checks from outside our container, but that means packaging the tooling we need into the Consul container. If we need to change the health check, then we end up re-deploying both our application and Consul, which unnecessarily couples the two.


### ContainerPilot to the rescue!

ContainerPilot is a shim written in Go to help make it easier to containerize existing applications. It can act as PID1 in the container and fork/exec the application. If the application exits then so does ContainerPilot.

Alternately, if your application double-forks (which is not recommended for containerized applications but hey we are taking about pre-container apps here!), you can run ContainerPilot as a side-by-side process within the container. In that case the container will not die if the application dies, which can create complicated failure modes but which can be mitigated by having a good TTL health check to detect the problem and alert you.

ContainerPilot registers the application with Consul on start and periodically sends TTL health checks to Consul; should the application fail then Consul will not receive the health check and once the TTL expires will no longer consider the application node healthy. Meanwhile, ContainerPilot runs background workers that poll Consul, checking for changes in dependent/upstream service, and calling an external executable on change.

Using local scripts to test health or act on backend changes means that we can run health checks that are specific to the service in the container, which keeps orchestration and the application bundled together.

ContainerPilot is explicitly *not* a supervisor process. Although it can act as PID1 inside a container, if the shimmed process dies, so does ContainerPilot (and therefore the container itself). ContainerPilot will return the exit code of its shimmed process back to the Docker Engine or Triton, so that it appears as expected when you run `docker ps -a` and look for your exit codes. ContainerPilot also attaches stdout/stderr from your application to stdout/stderr of the container, so that `docker logs` works as expected.

## Configuring ContainerPilot

ContainerPilot takes a single file argument (or a JSON string) as its configuration. All trailing arguments will be treated as the executable to shim and that executable's arguments.

```bash
# configure via passing a file argument
$ containerpilot -config file:///opt/containerpilot/app.json myapp --args --for --my --app

# configure via environment variable
$ export CONTAINERPILOT=file:///opt/containerpilot/app.json
$ containerpilot myapp --args --for --my --app

```

The format of the JSON file configuration is as follows:

```json
{
  "consul": "consul:8500",
  "onStart": "/opt/containerpilot/onStart-script.sh {{.ENV_VAR_NAME}}",
  "logging": {
    "level": "INFO",
    "format": "default",
    "output": "stdout"
  },
  "stopTimeout": 5,
  "preStop": "/opt/containerpilot/preStop-script.sh",
  "postStop": "/opt/containerpilot/postStop-script.sh",
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
      "onChange": "/opt/containerpilot/reload-app.sh"
    },
    {
      "name": "app",
      "poll": 10,
      "onChange": "/opt/containerpilot/reload-app.sh"
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
        "check": ["/bin/sensor.sh"]
      }
    ]
  }
}
```

#### Service fields:

- `name` is the name of the service as it will appear in Consul. Each instance of the service will have a unique ID made up from `name`+hostname of the container.
- `port` is the port the service will advertise to Consul.
- `health` is the executable (and its arguments) used to check the health of the service.
- `interfaces` is an optional single or array of interface specifications. If given, the IP of the service will be obtained from the first interface specification that matches. (Default value is `["eth0:inet"]`)
- `address` is an optional field to specify a DNS address or IP that will override what is obtained from traversing network interfaces. This can be useful in special cases such as during testing or when using bridged networking.
- `poll` is the time in seconds between polling for health checks.
- `ttl` is the time-to-live of a successful health check. This should be longer than the polling rate so that the polling process and the TTL aren't racing; otherwise Consul will mark the service as unhealthy.
- `tags` is an optional array of tags. If the discovery service supports it (Consul does), the service will register itself with these tags.

#### Backend fields:

- `name` is the name of a backend service that this container depends on, as it will appear in Consul.
- `poll` is the time in seconds between polling for changes.
- `onChange` is the executable (and its arguments) that is called when there is a change in the list of IPs and ports for this backend.

#### Service Discovery Backends:

Must supply only one of the following

- `consul` configures discovery via [Hashicorp Consul](https://www.consul.io/). Expects `hostname:port` string. If you are communicating with Consul over TLS you may include the scheme (ex. `https://consul:8500`):

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

#### Logging Config (Optional):

The logging config adjust the output format and verbosity of ContainerPilot logs.

- `level` adjusts the verbosity of the messages output by containerpilot. Must be one of: `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`, `PANIC` (Default is `INFO`)
- `format` adjust the output format for log messages. Can be `default`, `text`, or `json` (Default is `default`)
- `output` picks the output stream for log messages. Can be `stderr` or `stdout` (Default is `stdout`)

Logging Format Examples:

`default` - go log package with [LstdFlags](https://golang.org/pkg/log/)

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

#### Telemetry (Optional):

If a `telemetry` option is provided, ContainerPilot will expose a [Prometheus](http://prometheus.io) HTTP client interface that can be used to scrape performance telemetry. The telemetry interface is advertised as a service to the discovery service similar to services configured via the `services` block. Each `sensor` for the telemetry service will run periodically and record values in the [Prometheus client library](https://github.com/prometheus/client_golang). A Prometheus server can then make HTTP requests to the telemetry endpoint.

Details of how to configure the telemetry endpoint and how the telemetry endpoint works can be found in the [telemetry README](https://github.com/joyent/containerpilot/blob/master/telemetry/README.md).


#### Other fields:

- `onStart` is the executable (and its arguments) that will be called immediately prior to starting the shimmed application. This field is optional. If the `onStart` handler returns a non-zero exit code, ContainerPilot will exit.
- `preStop` is the executable (and its arguments) that will be called immediately **before** the shimmed application exits. This field is optional. ContainerPilot will wait until this program exits before terminating the shimmed application.
- `postStop` is the executable (and its arguments) that will be called immediately **after** the shimmed application exits. This field is optional. If the `postStop` handler returns a non-zero exit code, ContainerPilot will exit with this code rather than the application's exit code.
- `stopTimeout` Optional amount of time in seconds to wait before killing the application. (defaults to `5`). Providing `-1` will kill the application immediately.

*Note that if you're using `curl` to check HTTP endpoints for health checks, that it doesn't return a non-zero exit code on 404s or similar failure modes by default. Use the `--fail` flag for curl if you need to catch those cases.*

#### Interface Specifications

The `interfaces` parameter allows for one or more specifications to be used when searching for the advertised IP. The first specification that matches stops the search process, so they should be ordered from most specific to least specific.

- `eth0` : Match the first IPv4 address on `eth0` (alias for `eth0:inet`)
- `eth0:inet6` : Match the first IPv6 address on `eth0`
- `eth0[1]` : Match the 2nd IP address on `eth0` (zero-based index)
- `10.0.0.0/16` : Match the first IP that is contained within the IP Network
- `fdc6:238c:c4bc::/48` : Match the first IP that is contained within the IPv6 Network
- `inet` : Match the first IPv4 Address (excluding `127.0.0.0/8`)
- `inet6` : Match the first IPv6 Address (excluding `::1/128`)

Interfaces and their IP addresses are ordered alphabetically by interface name, then by IP address (lexicographically by bytes).

**Sample Ordering**

- eth0 10.2.0.1 192.168.1.100
- eth1 10.0.0.100 10.0.0.200
- eth2 10.1.0.200 fdc6:238c:c4bc::1
- lo ::1 127.0.0.1

#### Commands & arguments

All executable fields, such as `onStart` and `onChange`, accept both a string or an array. If a string is given, the command and its arguments are separated by spaces; otherwise, the first element of the array is the command path, and the rest are its arguments.

**String Command**

```json
"health": "/usr/bin/curl --fail -s http://localhost/app"
```

**Array Command**

```json
"health": [
  "/usr/bin/curl",
  "--fail",
  "-s",
  "http://localhost/app"
]
```

#### Template Configuration

ContainerPilot configuration has template support. If you have an environment variable such as `FOO=BAR` then you can use `{{.FOO}}` in your configuration file and it will be substituted with `BAR`.

**Example Usage**

```json
{
  "consul": "consul:8500",
  "onStart": "/opt/containerpilot/onStart-script.sh {{.URL_TO_SERVICE}} {{.API_KEY}}",
}
```

_Note:  If you need more than just variable interpolation, check out the [Go text/template Docs](https://golang.org/pkg/text/template/)._

## Operating ContainerPilot

ContainerPilot accepts POSIX signals to change its runtime behavior. Currently, ContainerPilot accepts the following signals:

- `SIGUSR1` will cause ContainerPilot to mark its advertised service for maintenance. ContainerPilot will stop sending heartbeat messages to the discovery service. The discovery service backend's `MarkForMaintenance` method will also be called (in the default Consul implementation, this deregisters the node from Consul).
- `SIGTERM` will cause ContainerPilot to send `SIGTERM` to the application, and eventually exit in a timely manner (as specified by `stopTimeout`).
- `SIGHUP` will cause ContainerPilot to reload its configuration. `onChange`, `health`, `preStop`, and `postStop` handlers will operate with the new configuration. This forces all advertised services to be re-registered, which may cause temporary unavailability of this node for purposes of service discovery.

Delivering a signal to ContainerPilot is most easily done by using `docker exec` and relying on the fact that it is being used as PID1.

```bash
docker exec myapp_1 kill -USR1 1

```

Docker will automatically deliver a `SIGTERM` with `docker stop`, not when using `docker kill`.  When ContainerPilot receives a `SIGTERM`, it will propagate this signal to the application and wait for `stopTimeout` seconds before forcing the application to stop. Make sure this timeout is less than the docker stop timeout period or services may not deregister from the discovery service backend. If `-1` is given for `stopTimeout`, ContainerPilot will kill the application immediately with `SIGKILL`, but it will still deregister the services.

*Caveat*: If ContainerPilot is wrapped as a shell command, such as: `/bin/sh -c '/opt/containerpilot .... '` then `SIGTERM` will not reach ContainerPilot from `docker stop`.  This is important for systems like Mesos which may use a shell command as the entrypoint under default configuration.

## Contributing

Please report any issues you encounter with ContainerPilot or its documentation by [opening a Github issue](https://github.com/joyent/containerpilot/issues). Roadmap items will be maintained as [enhancements](https://github.com/joyent/containerpilot/issues?q=is%3Aopen+is%3Aissue+label%3Aenhancement). PRs are welcome on any issue.

## Examples

We've published a number of example applications demonstrating how ContainerPilot works.

- [Applications on autopilot](https://www.joyent.com/blog/applications-on-autopilot)
- [CloudFlare DNS and CDN with dynamic origins](https://github.com/tgross/triton-cloudflare)
- [Consul, running as an HA raft](https://github.com/misterbisson/triton-consul)
- [Couchbase](https://www.joyent.com/blog/couchbase-in-docker-containers)
- [ELK stack](https://github.com/tgross/triton-elk)
- [Mesos on Joyent Triton](https://www.joyent.com/blog/mesos-by-the-pound)
- [Nginx with dynamic upstreams](https://www.joyent.com/blog/dynamic-nginx-upstreams-with-containerpilot)
- [MySQL (Percona Server) with auto scaling and fail-over](https://www.joyent.com/blog/dbaas-simplicity-no-lock-in)
- [Node.js + Nginx + Couchbase](https://www.joyent.com/blog/how-to-dockerize-a-complete-application)
