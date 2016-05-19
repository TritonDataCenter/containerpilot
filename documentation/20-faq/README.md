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

ContainerPilot can run tasks before starting the main app/process in the container, but those startup tasks must successfully complete and `exit 0` before ContainerPilot will continue and start the main app. If you need to run multiple processes simultaneously, you should use a supervisor such as runit.

Further discussion of this topic can be found in [#157](https://github.com/joyent/containerpilot/issues/157).

### Is ContainerPilot a supervisor?

ContainerPilot is explicitly *not* a supervisor process. Although it can act as PID1 inside a container, if the shimmed process dies, so does ContainerPilot (and therefore the container itself). ContainerPilot will return the exit code of its shimmed process back to the Docker Engine or Triton, so that it appears as expected when you run `docker ps -a` and look for your exit codes. ContainerPilot also attaches stdout/stderr from your application to stdout/stderr of the container, so that `docker logs` works as expected.

### Can ContainerPilot restart my application without stopping the container?

No. ContainerPilot is not a supervisor, and cannot stop and restart applications for configuration updates.

Further discussion of this topic can be found in [#126](https://github.com/joyent/containerpilot/issues/126).

### Is ContainerPilot open source?

Yes. ContainerPilot is licensed under the Mozilla Public License 2.0.

### I've been told that multi-process containers are bad. Is that true?

There are a number of conveniences in a container that does just one thing (offers a single service from a single app). It means that starting the container is the same as starting the app in it, and sending a signal to that container to shut down is the same as stopping the application. These containers offer a number of advantages for "modern" applications and operations, including automation of the application lifecycle.

However, it's hardly the only "right" way to build a container. ContainerPilot is optimized for containers that offer a single service, but [it can be used in containers and VMs running any number of services](#where-does-containerpilot-run). ContainerPilot makes absolutely no judgements about how you architect your application.

### Does ContainerPilot support IPv6?

It's not 

### How can I get support?

Please see [support options](/containerpilot/docs/support).
