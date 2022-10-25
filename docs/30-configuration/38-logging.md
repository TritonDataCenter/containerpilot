# Logging

The optional logging config adjusts the output format and verbosity of ContainerPilot logs.

- `level` adjusts the verbosity of the messages output by containerpilot. Must be one of: `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`, `PANIC` (Default is `INFO`)
- `format` adjust the output format for log messages. Can be `default`, `text`, or `json` (Default is `default`)
- `output` picks the output stream for log messages. Can be `stderr` or `stdout` (Default is `stdout`)

There are two sources of log data with ContainerPilot. First, ContainerPilot logs information about its own state, such as when jobs fail to run or events are triggered. Please note that `DEBUG` logging includes every event that's emitted by every job, and this can be quite a lot of information.

The second source of logs are the processes run by jobs and their health checks. By default, ContainerPilot attaches to stdout and stderr for every process it starts, and streams these logs to the logging framework. These logs will be emitted at `INFO` level, with one log entry per line emitted to `stdout` or `stderr`. The child process should terminate its output with newlines. The user can turn off this behavior for a job or its health check by setting the `raw` field of its `logging` configuration to `true`.

Logging Format Examples:

`default` - Uses the Go stdlib [log](https://golang.org/pkg/log/) package but with [RFC3339](https://golang.org/pkg/time/#pkg-constants) date format. Note that this format does not include any annotations about the origin of the log line, including the log level.

```
2015-03-26T01:27:38-04:00 Started observing beach
2015-03-26T01:27:38-04:00 A group of walrus emerges from the ocean
2015-03-26T01:27:38-04:00 The group's number increased tremendously!
2015-03-26T01:27:38-04:00 Temperature changes
2015-03-26T01:27:38-04:00 It's over 9000!
2015-03-26T01:27:38-04:00 The ice breaks!
```

`text` - [logrus TextFormatter](https://github.com/sirupsen/logrus)

```
time="2015-03-26T01:27:38-04:00" level=debug msg="Started observing beach" animal=walrus number=8
time="2015-03-26T01:27:38-04:00" level=info msg="A group of walrus emerges from the ocean" animal=walrus size=10
time="2015-03-26T01:27:38-04:00" level=warning msg="The group's number increased tremendously!" number=122 omg=true
time="2015-03-26T01:27:38-04:00" level=debug msg="Temperature changes" temperature=-4
time="2015-03-26T01:27:38-04:00" level=panic msg="It's over 9000!" animal=orca size=9009
time="2015-03-26T01:27:38-04:00" level=fatal msg="The ice breaks!" err=&{0x2082280c0 map[animal:orca size:9009] 2015-03-26 01:27:38.441574009 -0400 EDT panic It's over 9000!} number=100 omg=true
exit status 1
```

`json` - [logrus JSONFormatter](https://github.com/sirupsen/logrus)

```
{"animal":"walrus","level":"info","msg":"A group of walrus emerges from the ocean","size":10,"time":"2014-03-10 19:57:38.562264131 -0400 EDT"}
{"level":"warning","msg":"The group's number increased tremendously!","number":122,"omg":true,"time":"2014-03-10 19:57:38.562471297 -0400 EDT"}
{"animal":"walrus","level":"info","msg":"A giant walrus appears!","size":10,"time":"2014-03-10 19:57:38.562500591 -0400 EDT"}
{"animal":"walrus","level":"info","msg":"Tremendously sized cow enters the ocean.","size":9,"time":"2014-03-10 19:57:38.562527896 -0400 EDT"}
{"level":"fatal","msg":"The ice breaks!","number":100,"omg":true,"time":"2014-03-10 19:57:38.562543128 -0400 EDT"}
```

Logging details here do not affect how the Docker daemon (or other container runtime) handles logging. [See this blog post for a narrative and examples of how to manage log output from the container](https://www.tritondatacenter.com/blog/docker-log-drivers).
