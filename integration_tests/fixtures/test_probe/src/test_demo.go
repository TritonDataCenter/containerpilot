package main

import (
	"log"
	"net/http"
)

// TestDemo tests that Containerbuddy registers all services and that
// Nginx has a working route to App
func TestDemo(args []string) bool {

	consul, err := NewConsulProbe()
	if err != nil {
		log.Println(err)
	}

	// Wait for 1 healthy 'app' service to be registered with consul
	if err = consul.WaitForServices("app", "", 1); err != nil {
		log.Println(err)
		return false
	}
	// Wait for 1 healthy 'nginx' service to be registered with consul
	if err = consul.WaitForServices("nginx", "", 1); err != nil {
		log.Println(err)
		return false
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
