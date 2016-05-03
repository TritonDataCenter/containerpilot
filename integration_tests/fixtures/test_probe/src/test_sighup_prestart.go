package main

import "log"

func TestSigHupPrestart(args []string) bool {
	if len(args) != 1 {
		log.Println("TestSigHupPrestart requires 1 argument")
		log.Println(" - containerID: docker container to kill")
		return false
	}

	// Prestart will fix our healthcheck and SIGHUP us a config reload

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
