# Lifecycle

ContainerPilot's core functionality is centered around the concepts of jobs, the events that trigger those jobs or that jobs emit, and watches for the external service discovery catalog (Consul).


## What is a job?

A ContainerPilot job is a user-defined process and rules for when to execute it, how to health check it, and how to advertise it to Consul. The rules are intended to allow for flexibility to cover nearly any type of process one might want to run. Some possible job configurations include:

- A long running application like a web server, which needs to be restarted if it crashes.
- A one-time setup task that runs at the start of a container's lifetime but never again afterwards.
- A periodic process that runs every few minutes or hours, such as a backup.
- A task that runs when some other event occurs, such as running only when another job has become healthy.

By default every job will emit events for the lifecycle of its process: when the process exits, and whether that process succeeded (exit code 0) or failed (any other exit code). A job can opt in to health checking, where a user-defined health check process will run inside the container periodically and emit an event when the job becomes healthy or unhealthy. Jobs can opt in to service discovery by indicating what port they listen on. If the job has service discovery it will be registered with Consul and will send healthy/unhealthy events to Consul; this allows other containers to get information about the job's status via a `watch`.


## What is an event?

An event is a message about a change in the state of the container and its jobs. Every ContainerPilot job receives all the events as an asynchronous but ordered stream of messages. The events include the start of the container, a change in the health of a job, the exit (successful or not) of each job, a change to the state of a `watch`, the expiration of timeouts, etc. (A full listing with detailed explanation of each is available in the [job configuration](./30-configuration.md#job-configuration) section.)

Events are internal to a single instance of ContainerPilot; they are not shared to other containers. The only way the information about a job's health can be seen by other containers is if a job has been configured for service discovery and the other containers have a `watch` for that information.


## What is a watch?

A watch is a configuration of a service to watch in Consul. The watch monitors the state of the service and emits events when the service becomes healthy, becomes unhealthy, or has a change in the number of instances. Note that a watch does not include a behavior; watches only emit the event so that jobs can consume that event.


## How do events trigger jobs?

By default, a job's `exec` process starts as soon as ContainerPilot has finished startup. Many jobs will want to have a configuration that determines some specific event to wait for, using the `when` field. This field expects a `source` (the name of the event source) and either `once` (trigger one-time only) or `each` (trigger each time this event happens).

For example, if we want a job to run only once `myDb` is healthy (but not each time its health changes), we might use the following configuration:

```json5
when: {
  source: "myDb",
  once: "healthy"
}
```

Whereas if we want a job to run every time the watch for `myDb` emits a "changed" event, we might use the following configuration:

```json5
when: {
  source: "watch.myDb",
  each: "changed"
}
```

A detailed explanation of the field values can be found in [job configuration](./30-configuration.md#job-configuration) section.


## How can jobs be ordered?

Each job processes events in the order received, but each job operates concurrently; they each run in their own goroutine (think "thread" if you're not familiar with golang). It is not possible with ContainerPilot to force an ordering of the execution of each event. Some events will happen as a result of timers, whereas other events will happen as jobs crash or fail. But every job will receive events in the same order as every other job.

You can force an ordering of jobs to run by using the `when` field and setting the events as needed. For example, if we have `jobA` that depends on `jobB` to be healthy, but `jobB` depends on `jobC` to be completed, we might use a configuration like the following:

```json5
{
  name: "jobA",
  when: {
    source: "jobB",
    once: "healthy"
  }
},
{
  name: "jobB",
  when: {
    source: "jobC",
    once: "exitSuccess"
  }
},
{
  name: "jobC"
},
```

Note that the order of the jobs in the configuration file doesn't matter. ContainerPilot doesn't need to understand the ordering either -- the order of jobs falls out of the chain of events you create.
