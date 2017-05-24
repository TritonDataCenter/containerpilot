# Telemetry

If a `telemetry` option is provided, ContainerPilot will expose a [Prometheus](http://prometheus.io) HTTP client interface that can be used to scrape performance telemetry. The telemetry interface is advertised as a service to Consul. Each `sensor` for the telemetry service will run periodically and record values in the [Prometheus client library](https://github.com/prometheus/client_golang). A Prometheus server can then make HTTP requests to the telemetry endpoint.

Configuration details follow, but [this blog post offers a usage example and narrative](https://www.joyent.com/blog/containerpilot-telemetry) for it.

The top-level telemetry configuration defines the telemetry HTTP endpoint. This endpoint will be advertised to Consul (or other discovery service) just as a typical ContainerPilot `service` block is. The service will be called `containerpilot` and will be served on the path `/metrics`. The telemetry service will send periodic heartbeats to the discovery service to identify that it is still operating. There is no user-defined health check for the telemetry service endpoint, and you don't need to configure the poll/TTL; it will send a 15 second heartbeat every 5 seconds.

A minimal configuration for ContainerPilot including telemetry might look like this:

```json5
{
  consul: "consul:8500",
  telemetry: {
    port: 9090,
    interfaces: ["eth0"],
    tags: ["tag1"],
    sensors: [
      {
        namespace: "my_namespace",
        subsystem: "my_subsystem",
        name: "my_events_count",
        help: "help text",
        type: "counter",
        interval: 5,
        exec: "/bin/sensor.sh",
        timeout: "5s"
      }
    ]
  }
}
```

The fields are as follows:

- `port` is the port the telemetry service will advertise to the discovery service. (Default value is 9090.)
- `interfaces` is an optional single or array of interface specifications. If given, the IP of the service will be obtained from the first interface specification that matches. (Default value is `["eth0:inet"]`)
- `tags` is an optional array of tags. If the discovery service supports it (Consul does), the service will register itself with these tags.
- `sensors` is an optional array of sensor configurations (see below). If no sensors are provided, then the telemetry endpoint will still be exposed and will show only telemetry about ContainerPilot internals.

## Sensor configuration

The `sensors` field is a list of user-defined sensors that the telemetry service will use to collect telemetry. Each time a sensor is polled, the user-defined `exec` executable will be run. The sensor can use the [HTTP control socket](./37-control-plane.md) to record the metric value, or the sensor can call `containerpilot -putmetric` to record the metric.

- `namespace`, `subsystem`, and `name` are the names that the Prometheus client library will use to construct the name for the telemetry. These three names are concatenated with underscores `_` to become the final name that is scraped recorded by Prometheus. In the example above the metric recorded would be named `my_namespace_my_subsystem_my_event_count`. You can leave off the `namespace` and `subsystem` values and put everything into the `name` field if desired; the option to provide these other fields is simply for convenience of those who might be generating ContainerPilot configurations programmatically. Please see the [Prometheus documents on naming](http://prometheus.io/docs/practices/naming/) for best practices on how to name your telemetry.
- `help` is the help text that will be associated with the metric recorded by Prometheus. This is useful for debugging by giving a more verbose description.
- `type` is the type of collector Prometheus will use (one of `counter`, `gauge`, `histogram` or `summary`). See [below](#Collector_types) for details.
- `interval` is the time in seconds between running the `check`.
- `exec` is the executable (and its arguments) that is called when it is time to perform a telemetry collection.
- `timeout` an optional value to wait before forcibly killing the `check` handler. Handlers killed in this way are terminated immediately (`SIGKILL`) without an opportunity to clean up their state. The minimum timeout is `1ms` (see the golang [`ParseDuration`](https://golang.org/pkg/time/#ParseDuration) docs for this format). This field is optional and defaults to be equal to the `poll` time.

An example of a good check script might be:

```bash
#!/bin/bash
# check free memory
val=$(free | awk -F' +' '/Mem/{print $3}')
./containerpilot -putmetric "free_memory=$val"
```

### Collector types

ContainerPilot supports all four of the [metric types](http://prometheus.io/docs/concepts/metric_types/) available in the Prometheus API. Briefly these are:

##### Counter

A cumulative metric that represents a single numerical value that only ever goes up. A typical use case for a counter is a count of the number of of certain events. The value returned by the sensor will be added to the counter for that metric.

##### Gauge

A metric that represents a single numerical value that can arbitrarily go up and down. A typical use case for a gauge might be a measurement of the current memory usage. The value returned by the sensor script will be set as the new value for the gauge metric.

##### Histogram

A count of observations in "buckets", along with the sum of all observed values. A typical use case might be request durations or response sizes. When the Prometheus server scrapes this telemetry endpoint, it will receive a list of buckets and their counts. For example:

```
namespace_subsystem_response_bucket{le="1"} 0
namespace_subsystem_response_bucket{le="2.5"} 0
namespace_subsystem_response_bucket{le="5"} 1
namespace_subsystem_response_bucket{le="10"} 2
namespace_subsystem_response_bucket{le="+Inf"} 2
```

This indicates that the collector has seen 2 events in total. One event had a value less than 5 (`le="5"`), whereas a second was less than 10.

##### Summary

A summary is similar to a histogram, but while it also provides a total count of observations and a sum of all observed values, it calculates quantiles over a sliding time window. For example:

```
namespace_subsystem_response_seconds_summary{quantile="0.5"} 0.3
namespace_subsystem_response_seconds_summary{quantile="0.9"} 0.5
namespace_subsystem_response_seconds_summary{quantile="0.99"} 2
```

This indicates that the 50th percentile response time is 0.3 seconds, the 90th percentile is 0.5 seconds, and the 99th percentile is 2 seconds.

Please see the Prometheus docs on [histograms](http://prometheus.io/docs/practices/histograms/) for best practices on when you should choose histograms vs summaries.
