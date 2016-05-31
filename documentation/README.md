Title: ContainerPilot Documentation
----
Text:

ContainerPilot is an application-centric micro-orchestrator. It automates many of the operational tasks related to configuring a container as you start it, re-configuring it as you scale it or other containers around it, health-checking, and at other times during the container's lifecycle. ContainerPilot is written in Go, and runs inside the container with the main application.

### Contents

- [Installation in a container](/containerpilot/docs/installation)
- [Configuration](/containerpilot/docs/configuration)
- [Service discovery](/containerpilot/docs/service-discovery)
- [Container lifecycle](/containerpilot/docs/lifecycle)
- [Health checks](/containerpilot/docs/health)
- [`preStart`, `preStop`, `postStop`](/containerpilot/docs/start-stop)
- [Signals and operations](/containerpilot/docs/signals)
- [Periodic tasks](/containerpilot/docs/tasks)
- [Telemetry](/containerpilot/docs/telemetry)
- [Frequently asked questions](/containerpilot/docs/faq)
- [Support](/containerpilot/docs/support)

### Where it's used

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
