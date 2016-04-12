package main

import (
	"fmt"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	etcd "github.com/coreos/etcd/client"
	"time"
)

const etcdAddress = "http://etcd:4001"

// EtcdProbe is a test probe for etcd
type EtcdProbe interface {
	WaitForServices(service string, tag string, count int) error
}

type etcdClient struct {
	Client etcd.Client
	API    etcd.KeysAPI
	Prefix string
}

// NewEtcdProbe creates a new EtcdProbe for testing etcd
func NewEtcdProbe() (EtcdProbe, error) {
	cfg := etcd.Config{Endpoints: []string{etcdAddress}}
	client, err := etcd.New(cfg)
	if err != nil {
		return nil, err
	}
	kapi := etcd.NewKeysAPI(client)
	return EtcdProbe(etcdClient{Client: client, API: kapi, Prefix: "/containerpilot"}), nil
}

// WaitForServices waits for the healthy services count to equal the count
// provided or it returns an error
func (c etcdClient) WaitForServices(service string, tag string, count int) error {

	maxRetry := 30
	retry := 0
	var err error
	key := fmt.Sprintf("%s/%s", c.Prefix, service)
	for ; retry < maxRetry; retry++ {
		if retry > 0 {
			time.Sleep(1 * time.Second)
		}
		_, err := c.API.Get(context.Background(), key,
			&etcd.GetOptions{Recursive: true})

		return err
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("Service %s (tag:%s) count != %d", service, tag, count)
}
