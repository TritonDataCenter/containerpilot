package main

import "log"

func TestSigUsr1Prestart(args []string) bool {
	if len(args) != 1 {
		log.Println("TestSigUsr1Prestart requires 1 argument")
		log.Println(" - containerID: docker container to kill")
		return false
	}

	docker, err := NewDockerProbe()
	if err != nil {
		log.Println(err)
	}

	// Prestart will SIGUSR1 us into maintenance
	// Send SIGUSR1 to get us back out of maintenance
	if err = docker.SendSignal(args[0], SigUsr1); err != nil {
		log.Println(err)
		return false
	}

	// Wait for app to be healthy
	consul, err := NewConsulProbe()
	if err != nil {
		log.Println(err)
	}
	if err = consul.WaitForServices("app", "", 1); err != nil {
		log.Printf("Expected app to be healthy after SIGUSR1: %s\n", err)
		return false
	}
	return true
}
