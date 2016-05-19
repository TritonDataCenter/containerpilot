Title: Periodic tasks
----
Text:

Tasks are commands that are run periodically. They are typically used to perform housekeeping such as incremental back-ups, or pushing metrics to systems that cannot collect metrics through service discovery like Prometheus.

A task accepts the following properties:

- `command` is the executable (and its arguments) that will run when the task executes.
- `frequency` is the time between executions of the task. Supports milliseconds, seconds, minutes. The frequency must be a positive non-zero duration with a time unit suffix. (Example: `60s`) Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. The minimum frequency is `1ms`
- `timeout` is the amount of time to wait before forcibly killing the task.  Tasks killed in this way are terminated immediately (`SIGKILL`) without an opportunity to clean up their state. This value is optional and defaults to the `frequency`. The minimum timeout is `1ms`
- `name` is a friendly name given to the task for logging purposes - this has no effect on the task execution. This value is optional, and defaults to the `command` if not given.

**Note on task frequency:** *Pick a frequency of 1s or longer*. Although the task configuration permits frequencies as fast as 1ms, the overhead of spawning a process and its lifecycle is likely to be anywhere from 2ms to 25ms. Your task may not be able to run at all, or it might always be killed before it gets any useful work done. 
