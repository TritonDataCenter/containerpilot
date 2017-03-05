Title: Triton integration
----
Text:

There are a number of cases where we've wanted for lifecycle hooks to be able to communicate information back to ContainerPilot, or to be otherwise able to alter the ContainerPilot environment for purposes of future handler events or the main application itself. Additionally, it's likely that we at Joyent will want the [RFD36](https://github.com/joyent/rfd/blob/master/rfd/0036/README.md) scheduler for Triton to be able to have an interface to communicate with containers and ContainerPilot itself, for which we can provide an example in ContainerPilot.

This control plane would be useful for event hooks as well. Telemetry sensor outputs will be consumed via a robust API instead of the existing brittle text scraping. Event hooks will also be able to set environment variables for other services and event hooks.

The proposal is in two parts. A read-only endpoint exposing the state of all services associated with this ContainerPilot instance will be exposed as a new endpoint in the Telemetry HTTP server. A second endpoint that accepts HTTP POST updates to the state will be implemented as a listener on a unix domain socket only (at least in the initial implementation).

### Telemetry endpoint

##### `GetStatus GET /status`

This API will expose the state of all services associated with thisContainerPilot instance as seen by ContainerPilot. The API endpoint also reports the dependencies of each service.

*Example Response*

```
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 328
{
  "services": [
    {
      "name": "nginx",
      "address": "192.168.1.100",
      "port": 80,
      "status": "healthy"
      "depends_on": ["backend1", "backend2"]
    },
    {
      "name": "consul_agent",
      "address": "localhost",
      "port": 8500,
      "status": "nonadvertised"
    }
  ]
}
```

### Control plane endpoint

##### `PutEnv POST /v3/env`

This API allows a hook to update the environment variables that ContainerPilot provides to lifecycle hooks. The body of the POST must be in JSON format. The keys will be used as the environment variable to set, and the values will be the values to set for those environment variables. The environment variables take effect for all future processes spawned and override any existing environment variables. Unsetting an variable is supporting by passing an empty string or `null` as the JSON value for that key. This API returns HTTP400 if the key is not a valid environment variable name, otherwise HTTP200 with no body.

*Example Request*

```
curl -XPOST \
    -d '{"ENV1": "value1", "ENV2": "value2", "ENV_TO_CLEAR": ""}' \
    --unix-socket /var/containerpilot.sock \
    http:/v3/env
```

##### `PutMetric POST /v3/metric`

This API allows a sensor hook to update Prometheus metrics. (This allows sensor hooks to do so without having to suppress their own logging, which is required under 2.x.) The body of the POST must be in JSON format. The keys will be used as the metric names to update, and the values will be the values to set/add for those metrics. The API will return HTTP400 if the metric is not one that ContainerPilot is configuring, otherwise HTTP200 with no body.

*Example Request*

```
curl -XPOST \
    -d '{"my_counter_metric": 2, "my_gauge_metric": 42.42}' \
    --unix-socket /var/containerpilot.sock \
    http:/v3/environ
```

##### `Reload POST /v3/reload`

This API allows a hook to force ContainerPilot to reload its configuration from file. This replaces the SIGHUP handler from 2.x and behaves identically: all pollables are stopped, the configuration file is reloaded, and the pollables are restarted without interfering with the services. This endpoint returns a HTTP200 with no body.

*Example Request*

```
curl -XPOST \
    --unix-socket /var/containerpilot.sock \
    http:/v3/reload
```

##### `MaintenanceMode POST /v3/maintenance/{enable|disable}`

This API allows a hook to toggle ContainerPilot's maintenance mode. This replaces the SIGUSR1 handler from 2.x and behaves identically: when maintenance mode is enabled via the `enable` endpoint, all health checks are stopped and the discovery backend is sent a message to deregister the services. Requests to `/v3/status` will show all services with `status: "maintenance"`.

When the `disable` endpoint is used, ContainerPilot will exit maintenance mode. Requests to enable or disable maintenance mode are idempotent; requesting `enable` twice enables maintenance mode and does nothing on the second request. This endpoint returns a HTTP200 with a JSON body reporting whether the request was an update.

*Example Request*

```
curl -XPOST \
    --unix-socket /var/containerpilot.sock \
    http:/v3/maintenance/enable
```

*Example Response*

```
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 21
{
  "updated": true
}
```


Related GitHub issues:

- [Expose ContainerPilot state thru telemetry](https://github.com/joyent/containerpilot/issues/154)
- [HTTP control plane](https://github.com/joyent/containerpilot/issues/244)
- [Ability to pass SIGUSR1 and SIGHUP to main app](https://github.com/joyent/containerpilot/issues/195)
- [SIGINT termination and coprocess suspension](https://github.com/joyent/containerpilot/pull/186)
