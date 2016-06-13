Title: Coprocesses
----
Text:

Coprocesses are processes that run alongside the main application. Unlike tasks or other lifecycle hooks, coprocesses remain running. Coprocesses are treated as "secondary" to the main application. The `SIGHUP`/`SIGUSR1` handlers for ContainerPilot don't forward these signals to the coprocess. The stdout/stderr of the coprocess is piped through the ContainerPilot logging system just as a `health` or `task` does. Coprocesses will be restarted if the `restarts` flag is set, but do not cause ContainerPilot to exit the way the main application does.

A coprocess accepts the following properties:

- `command` is the executable (and its arguments) that will run when the coprocess executes.
- `name` is a friendly name given to the coprocess for logging purposes - this has no effect on the coprocess execution. This value is optional, and defaults to the `command` if not given.
- `restarts` is the number of times a coprocess will be restarted if it exits. Supports any non-negative numeric value (ex. `0`, `1`) or the strings `"unlimited"` or `"never"`. This value is optional and defaults to `"never"`.

### Startup behavior

Coprocesses are started immediately following the exit of the `preStart` hook, before polling for `health` or `backend` hooks, and before the main application starts. Because the coprocess may take some non-zero time to be "live" after it starts (for example, it needs to load its config from disk), if the main application depends on the coprocess to be running you'll need to account for this to avoid race conditions between the coprocess and the main application.

Because coprocess arguments are configured in the ContainerPilot configuration, you can use environment variables to templatize the configuration just as you would for other lifecycle hooks.

### Configuration reload

If ContainerPilot receives `SIGHUP` it reloads its configuration as described in [Signals and operations](/containerpilot/docs/signals). All coprocesses are stopped when this happens. The restart limit for the coprocess is reset to the new `restarts` value, even if was unchanged. And then the new coprocesses are started with the new configuration.
