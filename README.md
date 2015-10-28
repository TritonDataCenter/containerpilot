# containerbuddy

*A service for assisting discovery and configuration of applications running in containers.*


### Container-native applications vs all the rest

Applications in containers typically need to talk to a source of truth to discover their upstream services and tell their downstream services where to find them. Container-native applications come into the world understanding this responsibility, but no one wants to rewrite all our current applications to do this.

We can wrap each application in a shell script that registers itself with the discovery service easily enough, but watching for changes to that service and ensuring that health checks are being made is more complicated. We can put a second process in the container, but unless we make a supervisor as PID1 then there's no way of knowing whether our buddy process has died.

Additionally, discovery services like Consul provide a means of performing health checks from outside our container, but that means packaging the tooling we need into the Consul container. If we need to change the health check, then we end up re-deploying both our application and Consul, which unnecessarily couples the two.


### containerbuddy to the rescue!

containerbuddy is a shim written in Go to help make it easier to containerize existing applications. It can act as PID1 in the container and fork/exec the application. If the application exits then so does containerbuddy.

Alternately, if your application double-forks (which is not recommended for containerized applications but hey we are taking about pre-container apps here!), you can run containerbuddy as a side-by-side buddy process within the container. In that case the container will not die if the application dies, which can create complicated failure modes but which can be mitigated by having a good TTL health check to detect the problem and alert you.

containerbuddy registers the application with Consul on start and periodically sends TTL health checks to Consul; should the application fail then Consul will not receive the health check and once the TTL expires will no longer consider the application node healthy. Meanwhile, containerbuddy runs background workers that poll Consul, checking for changes in dependent/upstream service, and calling an external executable on change.

Using local scripts to test health or act on backend changes means that we can run health checks that are specific to the service in the container, which keeps orchestration and the application bundled together.

containerbuddy is explicitly *not* a supervisor process. Although it can act as PID1 inside a container, if the shimmed process dies, so does containerbuddy (and therefore the container itself). containerbuddy will return the exit code of its shimmed process back to the Docker Engine or Triton, so that it appears as expected when you run `docker ps -a` and look for your exit codes. containerbuddy also attaches stdout/stderr from your application to stdout/stderr of the container, so that `docker logs` works as expected.

### Configuring containerbuddy

containerbuddy takes a single file argument (or a JSON string) as its configuration. All trailing arguments will be treated as the executable to shim and that executable's arguments.

```bash
# configure via passing a file argument
$ containerbuddy -config file:///opt/containerbuddy/app.json myapp --args --for --my --app

# configure via environment variable
$ export CONTAINERBUDDY=file:///opt/containerbuddy/app.json
$ containerbuddy myapp --args --for --my --app

```

The format of the JSON file configuration is as follows:

```json
{
  "consul": "consul:8500",
  "services": [
    {
      "name": "app",
      "port": 80,
      "health": "/usr/bin/curl --fail -s http://localhost/app",
      "publicIp": false,
      "poll": 10,
      "ttl": 30
    }
  ],
  "backends": [
    {
      "name": "nginx",
      "poll": 30,
      "onChange": "/opt/containerbuddy/reload-app.sh"
    },
    {
      "name": "app",
      "poll": 10,
      "onChange": "/opt/containerbuddy/reload-app.sh"
    }
  ]
}
```

Service fields:
- `name` is the name of the service as it will appear in Consul. Each instance of the service will have a unique ID made up from `name`+hostname of the container.
- `port` is the port the service will advertise to Consul.
- `health` is the executable (and its arguments) used to check the health of the service.
- `publicIp` is an optional boolean flag indicating whether the service should advertise its public IP, rather than its private IP (defaults to private/`false`).
- `poll` is the time in seconds between polling for health checks.
- `ttl` is the time-to-live of a successful health check. This should be longer than the polling rate so that the polling process and the TTL aren't racing; otherwise Consul will mark the service as unhealthy.

Backend fields:
- `name` is the name of a backend service that this container depends on, as it will appear in Consul.
- `poll` is the time in seconds between polling for changes.
- `onChange` is the executable (and its arguments) that is called when there is a change in the list of IPs and ports for this backend.

*Note that if you're using `curl` to check HTTP endpoints for health checks, that it doesn't return a non-zero exit code on 404s or similar failure modes by default. Use the `--fail` flag for curl if you need to catch those cases.*

### Roadmap

As yet unimplemented features of containerbuddy include:
- Accept a `kill -HUP` to force a reload of configuration on external changes.
- Allow polling or TTLs to be tuned live, maybe by querying a key in Consul.
- Allow configuration to include tags about the service to register in Consul (ex. "prod", "dev").
- Allow alternative service discovery backends other than Consul.
- Reaping health checks from Consul some time after an application has been scaled away; currently if an instance is removed intentionally (rather than simply failing health checks) it will not be automatically removed from Consul.

### Running the example

In the `examples` directory is a simple application demonstrating how containerbuddy works. In this application, an Nginx node acts as a reverse proxy for any number of upstream application nodes. The application nodes register themselves with Consul as they come online, and the Nginx application is configured with an `onChange` handler that uses `consul-template` to write out a new virtualhost configuration file and then fires an `nginx -s reload` signal to Nginx, which causes it to gracefully reload its configuration.

To try this example on your own:

1. [Get a Joyent account](https://my.joyent.com/landing/signup/) and [add your SSH key](https://docs.joyent.com/public-cloud/getting-started).
1. Install the [Docker Toolbox](https://docs.docker.com/installation/mac/) (including `docker` and `docker-compose`) on your laptop or other environment, as well as the [Joyent CloudAPI CLI tools](https://apidocs.joyent.com/cloudapi/#getting-started) (including the `smartdc` and `json` tools)
1. [Configure Docker and Docker Compose for use with Joyent](https://docs.joyent.com/public-cloud/api-access/docker):

```bash
curl -O https://raw.githubusercontent.com/joyent/sdc-docker/master/tools/sdc-docker-setup.sh && chmod +x sdc-docker-setup.sh
./sdc-docker-setup.sh -k us-east-1.api.joyent.com <ACCOUNT> ~/.ssh/<PRIVATE_KEY_FILE>
```

At this point you can run the example on Triton:

```bash
cd ./examples
./start -p example

```

or in your local Docker environment:

```bash
cd ./examples/nginx
./start -p example -f docker-compose-local.yml

```

Let's scale up the number of `app` nodes:

```bash
docker-compose -p example scale app=3
```

As the nodes launch and register themselves with Consul, you'll see them appear in the Consul UI. The web page that the start script opens refreshes itself every 5 seconds, so once you've added new application containers you'll start seeing the "This page served by app server: <container ID>" change in a round-robin fashion.
