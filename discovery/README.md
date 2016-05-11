# Service Discovery

ContainerPilot can support multiple, "pluggable" backends for service discovery. If you want to implement a new backend, you are awesome and we'd love to see it!

Here are the steps you'll need to take:

## Create your discovery package

Create a new package under this folder named after your service discovery backend. For example: If you are implementing ZooKeeper, you'd make a folder called `zookeeper`. Also name the `.go` files after the backend as well:

```
+ discovery/
|-+ zookeeper/
  |-- zookeeper.go
  |-- zookeeper_test.go
```

## Implement `discovery.ServiceBackend` interface

Create a struct that represents your backend and make sure that it defines all the required functions to be considered a `discovery.ServiceBackend`.

Create a function which accepts a raw `interface{}` and returns either your service discovery struct, or an error if there was a parsing problem. Check the other backends `consul` and `etcd` for an example. Also look at `utils.DecodeRaw` for a utility that can transform this raw value into a concrete type or struct.

Include unit and integration tests so that we can verify the implementation easily and detect breaking changes.

So far, we've seen two types of discovery backends:

### 1. Service Registry

A service registry will have first-class support for services. It allows registering the IP/Port of a service by name and most likely supports health checking, and DNS or other means of listing the healthy services. Consul is one such service registry.  If you are implementing a backend like this, look at `consul` as an example.

### 2. Coordinated Filesystem

A coordinated filesystem is less of a service discovery mechanism and more of a primitive on which you can build a service discovery protocol. Ideally, we would like to keep the ContainerPilot implementation consistent across these types of backends, so look at the `etcd` backend and try to mirror the implementation used there: the paths (eg: `/containerpilot/{serviceName}/{serviceID}`), and the document body (`etcd.ServiceNode`).

## Register the backend with `discovery.RegisterBackend`

You'll need to register the backend name and a config function so that it can be identified and handed-off to your backend creation function by `config.go`.

There are two steps involved with this:

First, create an init function to register the name of your backend and a `ServiceDiscoveryConfigHook`function for parsing the configuration object. Here is an example for ZooKeeper:

```go
// init registers your backend name and  function
func init() {
	discovery.RegisterBackend("zookeeper", ConfigHook)
}

// ConfigHook simply implements the correct function signature
func ConfigHook(raw interface{}) (discovery.DiscoveryService, error) {
	return NewZookeeper(raw)
}

// NewZookeeper creates a new service discovery backend for ZooKeeper
func NewZookeeper(config interface{}) (*ZooKeeper, error) {
 // ...
}
```

Then, import your service discovery package in `main.go`:

```go
  // Import backends so that they initialize
_ "github.com/joyent/containerpilot/discovery/consul"
_ "github.com/joyent/containerpilot/discovery/etcd"
_ "github.com/joyent/containerpilot/discovery/zookeeper"
```
