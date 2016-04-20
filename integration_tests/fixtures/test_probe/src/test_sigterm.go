package main

import "log"

// TestSigterm tests that containerpilot will deregister the service on a SIGTERM
func TestSigterm(args []string) bool {
	if len(args) != 1 {
		log.Println("TestSigterm requires 1 argument")
		log.Println(" - containerID: docker container to kill")
		return false
	}

	consul, err := NewConsulProbe()
	if err != nil {
		log.Println(err)
	}
	docker, err := NewDockerProbe()
	if err != nil {
		log.Println(err)
	}

	// Wait for 1 healthy 'app' service to be registered with consul
	if err = consul.WaitForServices("app", "", 1); err != nil {
		log.Println(err)
		return false
	}

	// Send it a SIGTERM
	if err = docker.SendSignal(args[0], SigTerm); err != nil {
		log.Println(err)
		return false
	}

	// Wait for zero healthy 'app' services
	if err = consul.WaitForServices("app", "", 0); err != nil {
		log.Println(err)
		return false
	}
	return true
}
