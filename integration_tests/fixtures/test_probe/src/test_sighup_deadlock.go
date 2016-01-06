package main

import "log"
import "time"
import "math/rand"

// TestSighupDeadlock regression test deadlock between
// SIGUSR1 and SIGHUP: See GH-73
func TestSighupDeadlock(args []string) bool {
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
		log.Printf("Expected app to be healthy before the test starts: %s\n", err)
		return false
	}

	// Send it a SIGHUP a lot to try and force a deadlock
	// 300x times with a random delay between each try
	for i := 0; i < 300; i++ {
		if err = docker.SendSignal(args[0], SigHup); err != nil {
			log.Println(err)
			return false
		}
		// Add some delay so that the config can reload
		time.Sleep(time.Duration(rand.Int63n(30)) * time.Millisecond)
	}

	// Wait for TTL to expire
	time.Sleep(6 * time.Second)

	// Wait for app to be healthy
	if err = consul.WaitForServices("app", "", 1); err != nil {
		log.Printf("Expected app to be healthy after SIGHUP: %s\n", err)
		return false
	}
	return true
}
