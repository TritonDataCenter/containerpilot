Title: Health checks
----
Text:

ContainerPilot help ensure overall application up-time by monitoring and reporting the health of its components. Each service can be monitored with a user-defined health check command. Any non-zero exit status is considered unhealthy.

Health checks can be as simple as a `curl`, such as in [our Nginx implementation](https://github.com/autopilotpattern/nginx/blob/master/etc/containerpilot.json):

```
"health": "/usr/bin/curl --fail -s http://localhost/health"
```

Health checks can also be far more sophisticated. The Autopilot Pattern MySQL implementation [specifies a health check in a Python script](https://github.com/autopilotpattern/mysql/blob/master/etc/containerpilot.json):

```
"health": "python /usr/local/bin/manage.py health"
```

That script does a lot of work, but [the health check does a simple MySQL query](https://github.com/autopilotpattern/mysql/blob/master/bin/manage.py) to make sure the node is running:

```
mysql_query(node.conn, 'SELECT 1', ())
```

**Note** if you're using `curl` to check HTTP endpoints for `health` checks, it doesn't return a non-zero exit code on 404s or similar failure modes by default. Use the `--fail` flag for curl if you need to catch those cases.
