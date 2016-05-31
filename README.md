# ContainerPilot

*An application-centric micro-orchestrator that automates the process of service discovery, configuration, and lifecycle management inside the container, so you can focus on your apps.*

[![Build Status](https://travis-ci.org/joyent/containerpilot.svg)](https://travis-ci.org/joyent/containerpilot)
[![MPL licensed](https://img.shields.io/badge/license-MPL_2.0-blue.svg)](https://github.com/joyent/containerpilot/blob/master/LICENSE)

## What is ContainerPilot?

ContainerPilot is an application-centric micro-orchestrator that automates the process of service discovery, configuration, and lifecycle management inside the container.

Orchestration is the automation of the operations of an application. Most application require operational tasks like connecting them to related components ([WordPress needs to know where it's MySQL and Memcached servers are, for example](https://www.joyent.com/blog/wordpress-on-autopilot)), and some applications require special attention as they start up or shut down to be sure they bootstrap correctly or persist their data. We can do all that by hand, but modern applications automate those tasks in code. That's called "orchestration."

To make this work, every application needs to do the following (at a minimum):

- Register itself in a service catalog (like Consul or Etcd) for use by other apps
- Look to the service catalog to find the apps it depends on
- Configure itself when the container starts, and reconfigure itself over time

We can write our new applications to do that, but existing apps will need some help. We can wrap each application in a shell script that registers itself with the discovery service easily enough, but watching for changes to that service and ensuring that health checks are being made is more complicated. We can put a second process in the container, but unless we make a supervisor as PID1 then there's no way of knowing whether our shimmed process has died.

### ContainerPilot to the rescue!

ContainerPilot is a helper written in Go to make it easier to containerize our applications. It can act as PID1 in the container and fork/exec the application. If the application exits then so does ContainerPilot. ContainerPilot reaps zombies, runs health checks, registers the app in the service catalog, watches the service catalog for changes, and runs your user-specified code at events in the lifecycle of the container to make it all work right.

ContainerPilot is explicitly *not* a supervisor process. Although it can act as PID1 inside a container, it can't manage multiple services. And, if the shimmed process dies, so does ContainerPilot (and therefore the container itself). ContainerPilot will return the exit code of its shimmed process back to the Docker Engine or Triton, so that it appears as expected when you run `docker ps -a` and look for your exit codes. ContainerPilot also attaches stdout/stderr from your application to stdout/stderr of the container, so that `docker logs` works as expected.

## Getting started

See the [ContainerPilot documentation](https://www.joyent.com/containerpilot/docs) to get started, or jump to specific sections:

- [Installation in a container](https://www.joyent.com/containerpilot/docs/installation)
- [Configuration](https://www.joyent.com/containerpilot/docs/configuration)
- [Service discovery](https://www.joyent.com/containerpilot/docs/service-discovery)
- [Container lifecycle](https://www.joyent.com/containerpilot/docs/lifecycle)
- [Health checks](https://www.joyent.com/containerpilot/docs/health)
- [`preStart`, `preStop`, `postStop`](https://www.joyent.com/containerpilot/docs/start-stop)
- [Signals and operations](https://www.joyent.com/containerpilot/docs/signals)
- [Periodic tasks](https://www.joyent.com/containerpilot/docs/tasks)
- [Telemetry](https://www.joyent.com/containerpilot/docs/telemetry)
- [Frequently asked questions](https://www.joyent.com/containerpilot/docs/faq)
- [Support](https://www.joyent.com/containerpilot/docs/support)

You might also read [our guide building self-operating applications with ContainerPilot](https://www.joyent.com/blog/applications-on-autopilot) and look at the examples below.

### Examples

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
- [Nginx with dynamic upstreams](https://www.joyent.com/blog/dynamic-nginx-upstreams-with-containerpilot)

## Contributing

Please report any issues you encounter with ContainerPilot or its documentation by [opening a Github issue](https://github.com/joyent/containerpilot/issues). Roadmap items will be maintained as [enhancements](https://github.com/joyent/containerpilot/issues?q=is%3Aopen+is%3Aissue+label%3Aenhancement). PRs are welcome on any issue.

Details about contributing to documentation are in [documentation/CONTRIBUTING.md](https://github.com/joyent/containerpilot/blob/master/documentation/CONTRIBUTING.md)