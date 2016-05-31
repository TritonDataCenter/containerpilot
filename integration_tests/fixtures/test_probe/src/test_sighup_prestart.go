package main

import "log"

// The health check in the containerpilot config is intentionally
// broken. The preStart script will fix the health check and then
// SIGHUP to perform a config reload.
func TestSigHupPrestart(args []string) bool {
	if len(args) != 1 {
		log.Println("TestSigHupPrestart requires 1 argument")
		log.Println(" - containerID: docker container to kill")
		return false
	}

	consul, err := NewConsulProbe()
	if err != nil {
		log.Println(err)
	}
	// Wait for 1 healthy 'app' service to be registered with consul
	if err = consul.WaitForServices("app", "", 1); err != nil {
		log.Printf("Expected app to be healthy after SIGHUP: %s\n", err)
		return false
	}
	return true
}
