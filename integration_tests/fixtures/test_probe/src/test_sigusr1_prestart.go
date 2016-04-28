package main

import "log"
import "time"
import "math/rand"

func TestSigUsr1Prestart(args []string) bool {
	if len(args) != 1 {
		log.Println("TestSigterm requires 1 argument")
		log.Println(" - containerID: docker container to kill")
		return false
	}

	docker, err := NewDockerProbe()
	if err != nil {
		log.Println(err)
	}

	// Prestart will SIGUSR1 us into maintenance
	// Send SIGHUP to reload and resume
	if err = docker.SendSignal(args[0], SigHup); err != nil {
		log.Println(err)
		return false
	}
	// Add some delay so that the config can reload
	time.Sleep(time.Duration(rand.Int63n(30)) * time.Millisecond)

	// Wait for app to be healthy
	consul, err := NewConsulProbe()
	if err != nil {
		log.Println(err)
	}
	if err = consul.WaitForServices("app", "", 1); err != nil {
		log.Printf("Expected app to be healthy after SIGHUP: %s\n", err)
		return false
	}
	return true
}
