package main

import "log"

// TestConsul test that consul has elected a leader
func TestConsul(args []string) bool {
	consul, err := NewConsulProbe()
	if err != nil {
		log.Printf("Expected to be able to create consul client before the test starts: %s\n", err)
		return false
	}

	err = consul.WaitForLeader()
	if err != nil {
		log.Printf("Expected consul to elect leader before the test starts: %s\n", err)
		return false
	}
	return true
}
