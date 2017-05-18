# Watches

A watch is a configuration of a service to watch in Consul. The watch monitors the state of the service and emits events when the service becomes healthy, becomes unhealthy, or has a change in the number of instances. Note that a watch does not include a behavior; watches only emit the event so that jobs can consume that event.

Watch configurations include only the following fields:

```json5
watches: [
  {
    name: "backend",
    interval: 3,
    tag: "prod" // optional
  }
]
```

The `interval` is the time (in seconds) between polling attempts to Consul. The `name` is the service to query and the `tag` is the optional tag to add to the query.

A watch keeps in memory a list of the healthy IP addresses associated with the service. The list is not persisted to disk and if ContainerPilot is restarted it will need to check back in with the canonical data store which is Consul. If this list changes between polls, the watch emits one or two events:

- A `changed` event is emitted whenever there is a change.
- A `healthy` event is emitted whenever the watched service becomes healthy. This might mean that the state was previously unknown (as when ContainerPilot first starts up) or that it was previously unhealthy and is now unhealthy. This event will only be fired once for each change in status or count of instances. Subsequent polls that return the same value will not emit the event again.
- A `unhealthy` event is emitted whenever the watched service becomes unhealthy. This might mean that the service is not yet running when we first poll, or that it was previously healthy and is now unhealthy. This event will only be fired once for each change of status. Subsequent polls that return the same value will not emit the event again.

The name of the events emitted by watches are namespaced so as not to collide with internal job names. These events are prefixed by `watch.`. Here is an example configuration for a job listening for a watch event:

```json5
jobs: [
  {
    name: "update-app",
    exec: "/bin/update-app.sh",
    when: {
      source: "watch.backend",
      each: "changed"
    }
  }
],
watches: [
  {
    name: "backend",
    interval: 3
  }
]
```

In this example, the watch `backend` will be checked every 3 seconds. Each time the watch emits the `changed` event, the `update-app` job will execute `/bin/update-app.sh`.
