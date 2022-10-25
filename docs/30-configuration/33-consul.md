# Consul

ContainerPilot uses Hashicorp's [Consul](https://www.consul.io/) to register jobs in the container as services. Watches look to Consul to find out the status of other services.

## Client configuration

The `consul` field in the ContainerPilot config file configures ContainerPilot's Consul client. For use with Consul's ACL system, use the `CONSUL_HTTP_TOKEN` environment variable. If you are communicating with Consul over TLS you may include the scheme (ex. https://consul:8500). Note that generally the Consul client will be communicating to an agent on localhost, so TLS may not be necessary. If you need extra configuration options for TLS, you can use the following optional fields (or environment variable options described in the [Consul documentation](https://www.consul.io/docs/commands/index.html#environment-variables)) instead of a simple string:

```json5
consul: {
  address: "consul.example.com:8500",
  scheme: "https",
  token: "aba7cbe5-879b-999a-07cc-2efd9ac0ffe", // or CONSUL_HTTP_TOKEN
  tls: {
    cafile: "ca.crt",                 // or CONSUL_CACERT
    capath: "ca_certs/",              // or CONSUL_CAPATH
    clientcert: "client.crt",         // or CONSUL_CLIENT_CERT
    clientkey: "client.key",          // or CONSUL_CLIENT_KEY
    servername: "consul.example.com", // or CONSUL_TLS_SERVER_NAME
    verify: true,                     // or CONSUL_HTTP_SSL_VERIFY
  }
}
```

## Consul agent configuration

In a typical application deployment such as on Joyent's Triton [infrastructure containers](https://docs.tritondatacenter.com/public-cloud/instances/infrastructure) or in virtual machines, the end user will deploy a Consul agent onto each host (infrastructure container or VM). All applications on that same host will find that agent at localhost on the host or via bridge networking.

This does not work in environments like [Triton Elastic Docker Host](https://docs.tritondatacenter.com/public-cloud/instances/docker) or other container-as-a-service deployments where the end user can't deploy host-local services. In this kind of deployment, the user might try to deploy a Consul agent on each underlying host (using whatever host affinity options are provided by the deployment API), but the containers typically won't have a way of finding that agent.

In this scenario we recommend deploying a Consul agent inside each container. All processes (including ContainerPilot) can talk to the agent via localhost, and the agent can find the Consul servers via infrastructure-backed DNS (such as [Triton CNS](https://docs.tritondatacenter.com/public-cloud/network/cns)).

A suggested configuration of the Consul client and the job for the Consul agent is as follows, assuming that the environment variable `CONSUL` stands for the DNS name of the Consul servers:

```json5
consul: "localhost:8500",
jobs: [
  {
    name: "consul-agent",
    exec: [
        "consul", "agent",
        "-config-file=/etc/consul/consul.json",
        "-rejoin",
        "-retry-join", "{{ .CONSUL }}",
        "-retry-max", "10",
        "-retry-interval", "10s"
      ],
      restarts: "unlimited"
  }
]
```
