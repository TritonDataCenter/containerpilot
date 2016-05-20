Title: preStart, preStop, & postStop
----
Text:

Many applications require specific conditions to be set just before startup. For example, Nginx requires a configuration file that specifies each back end it connects to, but those back ends need to be resolved at run-time. That's ideal for the `preStart` event. [Here's an example](https://github.com/autopilotpattern/nginx/blob/master/etc/containerpilot.json):

```bash
"preStart": "/usr/local/bin/reload.sh preStart",
```

That command executes the `preStart` function in [the `reload.sh` script](https://github.com/autopilotpattern/nginx/blob/master/bin/reload.sh):

```bash
# Render Nginx configuration template using values from Consul,
# but do not reload because Nginx has't started yet
preStart() {
    consul-template \
        -once \
        -dedup \
        -consul ${CONSUL}:8500 \
        -template "/etc/nginx/nginx.conf.ctmpl:/etc/nginx/nginx.conf"
}
```

That command string uses consul-template to generate a configuration file from a template using details about the back-ends from Consul.

[A proposed improvement to the Autopilot Pattern Couchbase implementation](https://github.com/autopilotpattern/couchbase/issues/14) would automatically remove a node from the cluster after [receiving the `SIGTERM`](/containerpilot/docs/signals), but before stopping the Couchbase service in the container using `preStop`.