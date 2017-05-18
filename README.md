# ContainerPilot

*An init system for cloud-native distributed applications that automates the process of service discovery, configuration, and lifecycle management inside the container, so you can focus on your apps.*

[![Build Status](https://travis-ci.org/joyent/containerpilot.svg)](https://travis-ci.org/joyent/containerpilot)
[![MPL licensed](https://img.shields.io/badge/license-MPL_2.0-blue.svg)](https://github.com/joyent/containerpilot/blob/master/LICENSE)

## What is ContainerPilot?

Orchestration is the automation of the operations of an application. Most application require operational tasks like connecting them to related components ([WordPress needs to know where it's MySQL and Memcached servers are, for example](https://www.joyent.com/blog/wordpress-on-autopilot)), and some applications require special attention as they start up or shut down to be sure they bootstrap correctly or persist their data. We can do all that by hand, but modern applications automate those tasks in code. That's called "orchestration."

To make this work, every application needs to do the following (at a minimum):

- Register itself in a service catalog (like Consul or Etcd) for use by other apps
- Look to the service catalog to find the apps it depends on
- Configure itself when the container starts, and reconfigure itself over time

We can write our new applications to do that, but existing apps will need some help. We can wrap each application in a shell script that registers itself with the discovery service easily enough, but watching for changes to that service and ensuring that health checks are being made is more complicated. We can put a second process in the container, but as soon as we do that we need an init system running inside the container as well.

### ContainerPilot to the rescue!

ContainerPilot is an init system designed to live inside the container. It acts as a process supervisor, reaps zombies, run health checks, registers the app in the service catalog, watches the service catalog for changes, and runs your user-specified code at events in the lifecycle of the container to make it all work right. ContainerPilot uses Consul to coordinate global state among the application containers.

## Quick Start Guide

Check out our ["Hello, World" application](https://github.com/autopilotpattern/hello-world) on GitHub. Assuming you have Docker and Docker Compose available, it's as easy as:

```
git clone git@github.com:autopilotpattern/hello-world.git
cd hello-world
docker-compose up -d
open http://localhost
```

This application blueprint demonstrates using ContainerPilot to update Nginx upstream configuration at runtime. Try scaling up via `docker-compose scale hello=2 world=3` to see the Nginx configuration updated.

You can also [download](https://github.com/joyent/containerpilot/releases) the latest release of ContainerPilot from GitHub.

## Documentation

Documentation for ContainerPilot and where it fits with the rest of the Triton ecosystem can be found at [www.joyent.com/containerpilot](https://www.joyent.com/containerpilot). The index below links to the documentation in this repo for convenience.

[Lifecycle](./docs/10-lifecycle.md)
- [What is a job?](./docs/10-lifecycle.md#what-is-a-job)
- [What is an event?](./docs/10-lifecycle.md#what-is-an-event)
- [What is a watch?](./docs/10-lifecycle.md#what-is-a-watch)
- [How do events trigger jobs?](./docs/10-lifecycle.md#how-do-events-trigger-jobs)
- [How can jobs be ordered?](./docs/10-lifecycle.md#how-can-jobs-be-ordered)

[Design: the Why of ContainerPilot](./docs/20-design.md)
- [Why active service discovery?](./docs/20-design.md#why-active-service-discovery)
- [Why are behaviors specified by application developer?](./docs/20-design.md#why-are-behaviors-specified-by-application-developer)
- [Why are jobs not the same as services?](./docs/20-design.md#why-are-jobs-not-the-same-as-services)
- [Why don't watches have behaviors?](./docs/20-design.md#why-dont-watches-have-behaviors)
- [Why isn't there a "post-start" event?](./docs/20-design.md#why-isnt-there-a-post-start-event)
- [Why should you not use ContainerPilot?](./docs/20-design.md#why-should-you-not-use-containerpilot)

[Configuration](./docs/30-configuration.md)
- [Installation](./docs/30-configuration.md#installation)
- [Configuration syntax](./docs/30-configuration.md#configuration-syntax)
- [Environment variable parsing and template rendering](./docs/30-configuration.md#environment-variable-parsing-and-template-rendering)
- [Consul configuration](./docs/30-configuration.md#consul-configuration)
  - [Client configuration](./docs/30-configuration.md#client-configuration)
  - [Consul agent configuration](./docs/30-configuration.md#consul-agent-configuration)
- [Job configuration](./docs/30-configuration.md#job-configuration)
  - [Service discovery](./docs/30-configuration.md#service-discovery)
  - [Health checks](./docs/30-configuration.md#health-checks)
  - [Restart behavior](./docs/30-configuration.md#restart-behavior)
  - [Pre-stop/post-stop behaviors](./docs/30-configuration.md#pre-stop-post-stop-behaviors)
- [Watch configuration](./docs/30-configuration.md#watch-configuration)
- [Telemetry configuration](./docs/30-configuration.md#telemetry-configuration)
  - [Sensor configuration](./docs/30-configuration.md#sensor-configuration)
- [Control plane](./docs/30-configuration.md#control-plane)
  - [ContainerPilot subcommands](./docs/30-configuration.md#containerpilot-subcommands)
- [Logging](./docs/30-configuration.md#logging)

[Support](./docs/40-support.md)
- [Where to file issues](./docs/40-support.md#where-to-file-issues)
- [Contributing](./docs/40-support.md#contributing)
- [Backwards compatibility](./docs/40-support.md#backwards-compatibility)

You might also read [our guide building self-operating applications with ContainerPilot](https://www.joyent.com/blog/applications-on-autopilot) and look at the examples below.

## Examples

We've published a number of example applications demonstrating how ContainerPilot works.

- [Applications on autopilot: a guide to how to build self-operating applications with ContainerPilot](https://www.joyent.com/blog/applications-on-autopilot)
- [MySQL (Percona Server) with auto scaling and fail-over](https://www.joyent.com/blog/dbaas-simplicity-no-lock-in)
- [Autopilot Pattern WordPress](https://www.joyent.com/blog/wordpress-on-autopilot)
- [ELK stack](https://www.joyent.com/blog/docker-log-drivers)
- [Node.js + Nginx + Couchbase](https://www.joyent.com/blog/docker-nodejs-nginx-nosql-autopilot)
- [CloudFlare DNS and CDN with dynamic origins](https://github.com/autopilotpattern/cloudflare)
- [Consul, running as an HA raft](https://github.com/autopilotpattern/consul)
- [Couchbase](https://github.com/autopilotpattern/couchbase)
- [Mesos on Joyent Triton](https://www.joyent.com/blog/mesos-by-the-pound)
- [Nginx with dynamic upstreams](https://www.joyent.com/blog/dynamic-nginx-upstreams-with-containerbuddy)
