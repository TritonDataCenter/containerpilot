Title: FAQ
----
Text:

### Where does ContainerPilot run?

Typically, ContainerPilot runs inside your container. It's designed to run as PID1 and fork/exec the primary application. If the container's primary application exits then so does ContainerPilot.

Alternately, if your application double-forks (which is not recommended for containerized applications but hey, we are taking about pre-container apps here!), you can run ContainerPilot as a side-by-side process within the container. In that case, signal handling may get confusing, and the container won't die if the application dies.

tl;dr: ContainerPilot is designed for Docker-style containers, but it can run anywhere, including in multi-process containers, infrastructure containers, and VMs.

### I want to use Etcd (or ZooKeeper), but all the examples demonstrate Consul

The service catalog that ContainerPilot looks up and registers services in is pluggable. Consul and Etcd are supported now, there is [discussion of ZooKeeper as well](https://github.com/joyent/containerpilot/issues/142).

### Can ContainerPilot start X before starting my main app?

ContainerPilot can run tasks before starting the main app/process in the container, but those startup tasks must successfully complete and `exit 0` before ContainerPilot will continue and start the main app. If you need to run multiple processes simultaneously, you should use a supervisor such as runit or a coprocess.

Further discussion of this topic can be found in [containerpilot#157](https://github.com/joyent/containerpilot/issues/157).

### Is ContainerPilot a supervisor?

ContainerPilot is explicitly *not* a supervisor process. Although it can act as PID1 inside a container, if the shimmed process dies, so does ContainerPilot (and therefore the container itself). ContainerPilot will return the exit code of its shimmed process back to the Docker Engine or Triton, so that it appears as expected when you run `docker ps -a` and look for your exit codes. ContainerPilot also attaches stdout/stderr from your application to stdout/stderr of the container, so that `docker logs` works as expected.

### Can ContainerPilot restart my application without stopping the container?

No. ContainerPilot is not a supervisor, and cannot stop and restart applications for configuration updates.

Further discussion of this topic can be found in [containerpilot#126](https://github.com/joyent/containerpilot/issues/126).

### I've been told that multi-process containers are bad. Is that true?

There are a number of conveniences in a container that does _just one thing_ (offers a single service from a single app). It means that starting the container is the same as starting the app in it, and sending a signal to that container to shut down is the same as stopping the application. These containers offer a number of advantages for "modern" applications and operations, including automation of the application lifecycle.

However, it's hardly the only "right" way to build a container. ContainerPilot is optimized for containers that offer a single service, but [it can be used in containers and VMs running any number of services](#where-does-containerpilot-run). ContainerPilot makes absolutely no judgments about how you architect your application.

### Does ContainerPilot support IPv6?

Please follow [containerpilot#52](https://github.com/joyent/containerpilot/issues/52) for details.

### How do I use ContainerPilot with Consul on Triton or on a PaaS?

Consul nodes can be either agents or servers. The architecture of Consul assumes that you have an agent local to each instance of a service, such as on the same VM or machine as other containers. On Triton or in a PaaS or "serverless" environment, this assumption doesn't quite work and so you may need to make some adjustments. Here are some options:

- Run a Consul agent as a [coprocess](/containerpilot/docs/coprocesses). Each container acts as its own Consul agent and will send status changes to the Consul server cluster. An example of this can be found [here](https://github.com/tgross/nginx-autopilotpattern/tree/coprocess). This keeps orchestration simple but adds a small amount of complexity to your container.
- Deploy a multi-process container with both your application and a Consul agent, with a supervisor like [runit](http://smarden.org/runit/) or [s6](http://skarnet.org/software/s6/). Each container acts as its own Consul agent and will send status changes to the Consul server cluster. An example of this can be found [here](https://github.com/tgross/nginx-autopilotpattern/tree/multiprocess). This keeps orchestration simple but adds a small amount of complexity to your container.
- Use a single Consul node as a "master" for writing. The failure of this node will prevent your services from getting updates for new members or failed members, but the services will otherwise operate normally. This lets you keep your container simple and gives you service availability at the cost of control plane availability.
- Divide instances of your services among your Consul nodes, so that if (for example) you are using 3 Consul nodes then you'll send a third of each service's traffic to each node. This lets you keep your container simple at the cost of scheduling and placement complexity.
- Use etcd. Although Consul has a richer API and better scalability, etcd does not assume that you're running local agents and so it might be a better solution for smaller deployments.

Please follow [containerpilot#162](https://github.com/joyent/containerpilot/issues/162) for more details.

### Is ContainerPilot open source?

Yes. ContainerPilot is licensed under the Mozilla Public License 2.0.

### How can I get support?

Please see [support options](/containerpilot/docs/support).
