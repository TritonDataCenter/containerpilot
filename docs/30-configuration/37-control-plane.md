# Control plane

Jobs often need a way to send information back to ContainerPilot to reload its own configuration, to update metrics, to put a service into maintenance mode, etc. ContainerPilot exposes a HTTP control plane that listens on a local unix socket. By default this can be found at `/var/run/containerpilot.socket`, and the location can be changed via the `control` configuration field.

### ContainerPilot subcommands

Because not all containers will include an HTTP client, ContainerPilot provides subcommands which can be used to send HTTP POSTs to the various control plane endpoints described below. A list of all subcommands can be found by invoking `-help`:

```
./containerpilot -help
Usage of ./containerpilot:
  -config string
        File path to JSON5 configuration file. Defaults to CONTAINERPILOT env var.
  -maintenance string
        Toggle maintenance mode for a ContainerPilot process through its control socket.
        Options: '-maintenance enable' or '-maintenance disable'
  -out string
        File path where to save rendered config file when '-template' is used.
        Defaults to stdout ('-').
  -ping
        Check that the ContainerPilot control socket is up.
  -putenv value
        Update environ of a ContainerPilot process through its control socket.
        Pass environment in the format: 'key=value'
  -putmetric value
        Update metrics of a ContainerPilot process through its control socket.
        Pass metrics in the format: 'key=value'
  -reload
        Reload a ContainerPilot process through its control socket.
  -template
        Render template and quit.
  -version
        Show version identifier and quit.
```

##### `PutEnv POST /v3/env`

This API allows a client to update the environment variables that ContainerPilot provides to jobs and health checks. The body of the POST must be in JSON format. The keys will be used as the environment variable to set, and the values will be the values to set for those environment variables. The environment variables take effect for all future processes spawned and override any existing environment variables. Unsetting an variable is supporting by passing an empty string or `null` as the JSON value for that key. This API returns HTTP400 if the key is not a valid environment variable name, otherwise HTTP200 with no body.

*Example Subcommand*

```
./containerpilot -putenv 'ENV1=value1' -putenv 'ENV2=value2' -putenv 'ENV_TO_CLEAR'
```

*Example HTTP Request*

```
curl -XPOST \
    -d '{"ENV1": "value1", "ENV2": "value2", "ENV_TO_CLEAR": ""}' \
    --unix-socket /var/containerpilot.sock \
    http:/v3/env
```

##### `PutMetric POST /v3/metric`

This API allows a client to update Prometheus metrics. The body of the POST must be in JSON format. The keys will be used as the metric names to update, and the values will be the values to set/add for those metrics. The API will return HTTP400 if the metric is not one that ContainerPilot is configuring, otherwise HTTP200 with no body.

*Example Subcommand*

```
./containerpilot -putmetric 'my_counter_metric=2' -putmetric 'my_gauge_metric=42.42'
```

*Example HTTP Request*

```
curl -XPOST \
    -d '{"my_counter_metric": 2, "my_gauge_metric": 42.42}' \
    --unix-socket /var/containerpilot.sock \
    http:/v3/environ
```

##### `Reload POST /v3/reload`

This API allows a client to force ContainerPilot to reload its configuration from file. This replaces the SIGHUP handler from 2.x and behaves identically: all pollables are stopped, the configuration file is reloaded, and the pollables are restarted without interfering with the services. This endpoint returns a HTTP200 with no body.

*Example Subcommand*

```
./containerpilot -reload
```

*Example HTTP Request*

```
curl -XPOST \
    --unix-socket /var/run/containerpilot.sock \
    http:/v3/reload
```

##### `MaintenanceMode POST /v3/maintenance/{enable|disable}`

This API allows a process to toggle ContainerPilot's maintenance mode. When maintenance mode is enabled via the `enable` endpoint, all health checks are stopped and the discovery backend is sent a message to deregister the services.

When the `disable` endpoint is used, ContainerPilot will exit maintenance mode. Requests to enable or disable maintenance mode are idempotent; requesting `enable` twice enables maintenance mode and does nothing on the second request. This endpoint returns a HTTP200 with a JSON body reporting whether the request was an update.

*Example Subcommand*

```
./containerpilot -maintenance=enable
```

*Example HTTP Request*

```
curl -XPOST \
    --unix-socket /var/run/containerpilot.sock \
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


##### `Ping GET /v3/ping`

This API checks if the ContainerPilot socket is up without mutating any state. This endpoint returns a HTTP200 if the socket is up.

*Example Subcommand*

```
./containerpilot -ping
```

*Example HTTP Request*

```
curl --unix-socket /var/run/containerpilot.sock \
    http:/v3/ping
```

*Example Response*

```
HTTP/1.1 200 OK
Content-Length: 2
ok
```
