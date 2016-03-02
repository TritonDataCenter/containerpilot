package main

import (
	"fmt"
	"log"
	"net/http"
)

// TestDiscovery tests that Containerbuddy registers all services and that
// Nginx has a working route to App
func TestDiscovery(args []string) bool {

	if len(args) == 1 && args[0] == "etcd" {
		if err := waitForEtcd(); err != nil {
			log.Printf("Failed to start test w/ etcd: %v\n", err)
			return false
		}
	} else {
		if err := waitForConsul(); err != nil {
			log.Printf("Failed to start test w/ consul: %v\n", err)
			return false
		}
	}

	if resp, err := http.Get("http://nginx:80/app/"); err != nil {
		log.Println(err)
		return false
	} else {
		if resp.StatusCode != 200 {
			log.Printf("/app returned %v\n", resp.Status)
			return false
		}
	}

	return true
}

// Wait for 1 healthy 'app' service and 1 healthy 'nginx' service
// to be registered with consul
func waitForConsul() error {
	if consul, err := NewConsulProbe(); err != nil {
		return fmt.Errorf("could not reach Consul: %v\n", err)
	} else {
		if err := consul.WaitForServices("app", "", 1); err != nil {
			return fmt.Errorf("app did not become healthy: %v\n", err)
		} else {
			if err = consul.WaitForServices("nginx", "", 1); err != nil {
				return fmt.Errorf("nginx did not become healthy: %v\n", err)
			}
		}
	}
	return nil
}

// Wait for 1 healthy 'app' service and 1 healthy 'nginx' service
// to be registered with etcd
func waitForEtcd() error {
	if etcd, err := NewEtcdProbe(); err != nil {
		return fmt.Errorf("could not reach etcd: %v\n", err)
	} else {
		if err := etcd.WaitForServices("app", "", 1); err != nil {
			return fmt.Errorf("app did not become healthy: %v\n", err)
		} else {
			if err = etcd.WaitForServices("nginx", "", 1); err != nil {
				return fmt.Errorf("nginx did not become healthy: %v\n", err)
			}
		}
	}
	return nil
}
