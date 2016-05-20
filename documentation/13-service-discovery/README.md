Title: Service discovery
----
Text:

One of the most important features of ContainerPilot is how it can watch a service catalog for changes in the back-ends that your container depends on. If, and more often, when there are changes in those back ends, it triggers an `onChange` event so your application can reconfigure itself.

### Scaling an application

Let's say you have application X that depends on application Y. When you scale up Y, you have to reconfigure X so that it's aware of the additional instances of Y. When you specify `backends` to watch, ContainerPilot will trigger the `onChange` event whenever those backends change.

[Read more about how ContainerPilot and the Autopilot Pattern work with a scheduler to support scaling](https://www.joyent.com/blog/app-centric-micro-orchestration).

### High availability

Sometimes containers fail. You'll want to find out why in time, but you don't want your whole application to fail because of a problem in a single container. By combining [health checks](/containerpilot/docs/health) with service discovery, ContainerPilot can watch for changes in container health and trigger the `onChange` event so your app can reconfigure itself and route requests to different containers.

[Read more about how active discovery, such as that supported by ContainerPilot, improves application reliability](http://containersummit.io/articles/active-vs-passive-discovery).

### Examples

[The Nginx implementation](https://github.com/autopilotpattern/nginx/blob/master/etc/containerpilot.json) watches for a back end specified in an environment variable, and triggers a script for the `onChange` event:

```json
  "backends": [
    {
      "name": "{{ .BACKEND }}",
      "poll": 7,
      "onChange": "/usr/local/bin/reload.sh"
    }
  ],
```

That script executes the the following when called for that event:

```bash
# Render Nginx configuration template using values from Consul,
# then gracefully reload Nginx
onChange() {
    consul-template \
        -once \
        -dedup \
        -consul ${CONSUL}:8500 \
        -template "/etc/nginx/nginx.conf.ctmpl:/etc/nginx/nginx.conf:nginx -s reload"
}
```

A more sophisticated example is in the [Autopilot Pattern MySQL implementation](https://www.joyent.com/blog/dbaas-simplicity-no-lock-in), which uses the `onChange` event to manage cluster health and recover from a failed primary.