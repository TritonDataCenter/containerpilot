package main

import (
	"fmt"
	"log"
	"net/http"
)

// TestDiscovery tests that ContainerPilot registers all services and that
// Nginx has a working route to App
func TestDiscovery(args []string) bool {
	if err := waitForConsul(); err != nil {
		log.Printf("Failed to start test w/ consul: %v\n", err)
		return false
	}
	resp, err := http.Get("http://nginx:80/app/")
	if err != nil {
		log.Println(err)
		return false
	}
	if resp.StatusCode != 200 {
		log.Printf("/app returned %v\n", resp.Status)
		return false
	}
	return true
}

// Wait for 1 healthy 'app' service and 1 healthy 'nginx' service
// to be registered with consul
func waitForConsul() error {
	consul, err := NewConsulProbe()
	if err != nil {
		return fmt.Errorf("could not reach Consul: %v", err)
	}
	if err := consul.WaitForServices("app", "", 1); err != nil {
		return fmt.Errorf("app did not become healthy: %v", err)
	}
	if err = consul.WaitForServices("nginx", "", 1); err != nil {
		return fmt.Errorf("nginx did not become healthy: %v", err)
	}
	return nil
}
