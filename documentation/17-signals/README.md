Title: Signal handling
----
Text:

ContainerPilot accepts POSIX signals to change its runtime behavior. Currently, ContainerPilot accepts the following signals:

- `SIGUSR1` will cause ContainerPilot to mark its advertised service for maintenance. ContainerPilot will stop sending heartbeat messages to the discovery service. The discovery service backend's `MarkForMaintenance` method will also be called (in the default Consul implementation, this deregisters the node from Consul).
- `SIGTERM` will cause ContainerPilot to send `SIGTERM` to the application, and eventually exit in a timely manner (as specified by `stopTimeout`).
- `SIGHUP` will cause ContainerPilot to reload its configuration. `onChange`, `health`, `preStop`, and `postStop` handlers will operate with the new configuration. This forces all advertised services to be re-registered, which may cause temporary unavailability of this node for purposes of service discovery.

Delivering a signal to ContainerPilot is most easily done by using `docker exec` and relying on the fact that it is being used as PID1.

```bash
docker exec myapp_1 kill -USR1 1
```

Docker will automatically deliver a `SIGTERM` with `docker stop`, not when using `docker kill`.  When ContainerPilot receives a `SIGTERM`, it will propagate this signal to the application and wait for `stopTimeout` seconds before forcing the application to stop. Make sure this timeout is less than the docker stop timeout period or services may not deregister from the discovery service backend. If `-1` is given for `stopTimeout`, ContainerPilot will kill the application immediately with `SIGKILL`, but it will still deregister the services.

**Caveat**: If ContainerPilot is wrapped as a shell command, such as: `/bin/sh -c '/opt/containerpilot .... '` then `SIGTERM` will not reach ContainerPilot from `docker stop`.  This is important for systems like Mesos which may use a shell command as the entrypoint under default configuration.