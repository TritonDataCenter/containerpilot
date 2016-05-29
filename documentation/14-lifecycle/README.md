Title: Lifecycle
----
Text:

The magic that ContainerPilot adds to an application is an awareness of the application lifecycle. This makes it possible to do tasks before an application starts or after it stops, for example, as well as health checks and recurring tasks while it's running. Perhaps most importantly, ContainerPilot can also watch for new, stopped, or unhealthy containers and trigger updates so the application can re-configure itself as needed.

The container's lifecycle starts and stops with the scheduler, so [this guide to the relationship between the scheduler and ContainerPilot](https://www.joyent.com/blog/app-centric-micro-orchestration) may be useful.

## At container startup

### `preStart`

The executable (and its arguments) that will be called immediately **prior** to starting the shimmed application. This field is optional. If the `preStart` handler returns a non-zero exit code, ContainerPilot will exit. [Read more](/containerpilot/docs/start-stop).

### Main application start

This is the main application specified in the Dockerfile's `CMD` or `ENTRYPOINT`, or in the `docker run...` string.

## While the container is running

### `health`

The health check(s) specified for each service are repeated at the specified interval. [Read more](/containerpilot/docs/health).

### `onChange`

This user-specified command is run whenever ContainerPilot detects a change in the makeup of the specified back-ends.

### Recurring tasks

ContainerPilot can execute any number of recurring tasks during the life of the container at user-specified intervals. This is an optional feature. [Read more](/containerpilot/docs/tasks).

### Telemetry

Optional application performance information can be gather via `telemetry` `sensors` that are run periodically. [Read more](/containerpilot/docs/telemetry).

## When stopping the container

### `preStop` 

The executable (and its arguments) that will be called immediately **before** ContainerPilot stops the shimmed application. This happens when [ContainerPilot receives a `SIGTERM` signal](/containerpilot/docs/signals), such as from Docker during a `docker stop...`. This field is optional. ContainerPilot will wait until the `preStop` handler exits before terminating the shimmed application. Note that the Docker Engine or Triton will send a `SIGKILL` after the timeout on `docker stop` expires, so you'll want to configure that timeout to allow enough time for your `preStop` to expire. [Read more](/containerpilot/docs/start-stop).

### `postStop`

The executable (and its arguments) that will be called immediately **after** the shimmed application exits. This field is optional. If the `postStop` handler returns a non-zero exit code, ContainerPilot will exit with this code rather than the application's exit code. [Read more](/containerpilot/docs/start-stop).