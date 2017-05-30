# Design: the Why of ContainerPilot

## Why active service discovery?

The goal of service discovery in cloud-native applications is to configure dependencies for our applications and update them as changes occur without manual intervention. There are two major approaches to solving this problem:

- Insert a proxy between the application and the dependency. The proxy knows which dependency instances are available and implements load balancing among them.
- Add dependency load balancing and dynamic configuration to the application itself.

The first option is a common approach in the container ecosystem. It represents all the right reasons to outsource a problem: we get quick scalability with no need to modify the existing application. The proxy is a centralized place to manage access to dependencies, so all we have to do is point the client application at the proxy and go. The proxy itself can be as simple as HAProxy, or it can be a more complex proxy that can direct writes to a primary and reads to the read replicas. This doesn't entirely solve the discovery problem but at least it centralizes the solution.

But there are several problems with this approach. Proxies add extra network latency and overhead. Proxies add new points of failure; if the proxy itself goes offline or is overloaded, the whole system fails instead of giving us a means to degrade gracefully with application-specific logic.

More generally, proxies can't provide application-level guarantees. Consider the case of a database with a primary that accepts writes and many read-replicas. The proxy might be able to direct queries to multiple replicas, and we might even be able to setup a hot standby database primary. We might be able to lean on infrastructure providers to allow our primary and standby to share a virtual IP. This works well in the simple case because it means that we don't have to make any changes to our application to achieve production-grade availability and scalability.

But this deeply ties the logic of the application to the infrastructure, particularly in cases of stateful applications like databases. The proxy is unlikely to be able to cover the wide variety of possible application-level concerns and approaches: sharding on key spaces, serializability or linearizability of writes, design patterns like event sourcing or CQRS, or deployment scenarios designed to support zero-downtime schema changes in databases that don't otherwise support live migrations. To put it another way, the proxy can't solve the CAP theorem problems (or any of the other challenges of distributed computing) for your application. That's the job of the application developer.

The alternative to that pattern is for active discovery and configuration in the application. The application registers itself with a "service catalog" and queries this catalog for any external dependencies. We'll also need to health check the application and keep the service catalog up to date. Although the application developer now has more responsibility for the behavior of their application, they also have the ability to get much more visibility into failures and application-specific control over how to handle them.

The difference between the two options is a matter of where decisions are made. Passive discovery patterns are those that separate the application from the decisions, leaving the application passive in both the choice of what back ends to connect to, and passive in resolving failures that may result. Active discovery patterns move those decisions into the application so it can have an active role in choosing the backend and working around failures it may encounter.

By making the application an active participant in the discovery process, we can eliminate a layer of complexity, misdirection, and latency between the application and its backends and give us faster, more reliable, and more resilient applications.

For more on this topic, see Casey Bisson's article on the ContainerSummit website: [_Active vs. passive discovery in distributed applications_](https://containersummit.io/articles/active-vs-passive-discovery).


## Why isn't there a "post-start" or "started" event?

In theory it's possible for ContainerPilot to fire an event immediately after it `fork/exec`'s a job's process. But in practice this is not a useful event for an application developer and will cause hard-to-debug race conditions in your configuration. Processes like servers need to bind to an address before they can be considered "started." Other processes need to do other work during start like loading and interpreting byte code. A short-lived task might even complete all its work and exit before an event could be handled, leading to an unsolvable race condition.

If ContainerPilot were to emit a `started` event then the job that emits the event will almost never be ready to do work when the event is handled. This means that jobs that depend on that job will need to implement some kind of health checking and retry logic anyways, and we would gain nothing at the cost of making configuration very confusing and error prone. Jobs that are dependent on another job should listen for the `healthy` event from their dependencies.


## Why Consul and not etcd or Zookeeper?

Consul provides a number of higher-level capabilities than a simple KV store like etcd or ZK. Providing the ability to use these capabilities would mean either going with a least-common-denominator approach or having complex provider-specific configuration options for tagging interfaces, providing secure connection to the backend, and faster deregistration on shutdown, among others. Additionally, Consul has first-class support for multi-datacenter deployments.

The primary argument for supporting etcd rather than Consul is that Kubernetes and related projects are using it as their service discovery layer. But there's no particular reason that the scheduler's own consensus and membership store needs to be the same store used by applications. And even perhaps _should not_ be the same store, given that in most organizations the team responsible for application development will not be the same team responsible for running the deployment platform.


## Why are jobs not the same as services?

A job is the core abstraction for managing all processes in ContainerPilot. A service is a job that's been registered with Consul. But the end user will not necessarily want to register all processes in a container. Health checks, sensors, setup tasks, etc. are all processes that a container needs to run that are internal to the container.


## Why don't watches or metrics have an exec field?

By having ContainerPilot's `exec` fields all in jobs rather than in watches or metrics, you can configure more than one `exec` behavior when an event is emitted.

For watches, ContainerPilot provides only the `changed`, `healthy`, and/or `unhealthy` events. The application developer can decide whether to coalesce multiple concurrent events into a single event, whether to start a job and then wait for that job to complete before starting a different one, or whatever they require for their application.

Likewise, for metrics, a single job might take a measurement from the application environment but write several metrics after parsing that measurement. A good example of this is Nginx's `stub_status` module, which provides several numerical measurements to a single HTTP GET. A job designed as a sensor might take this result, do some math on some of the numbers, and then execute `containerpilot -putmetric` multiple times.


## Why use something other than ContainerPilot?

ContainerPilot is just one option for implementing the Autopilot Pattern.

This pattern creates responsibilities on the application that many legacy applications will be unable to fulfill, and ContainerPilot is designed first and foremost with supporting those kinds of applications. Greenfield applications in organizations that already have rich libraries and tooling for handling questions of service discovery, health checking, process supervision, metrics collection, etc. and that have their own service catalog already deployed will be unlikely to get much out of using ContainerPilot.

Much of what ContainerPilot does can be assembled from existing components. One can include in a container an init system and supervisor like [`s6`](http://skarnet.org/software/s6/), which runs Consul agent and [`consul-template`], a [Prometheus](https://prometheus.io/) metrics collection agent, a tool like [registrar](https://gliderlabs.com/registrator/latest/), and a collection of bash scripts to tie all the pieces together. ContainerPilot is an opinionated choice of tooling, but if your organization has strong feelings about those tools then it's possible to implement its behaviors by a sufficiently motivated development team.
