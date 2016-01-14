package main

import docker "github.com/samalba/dockerclient"

const dockerSocket = "unix:///var/run/docker.sock"

// DockerSignal is a signal that can be sent to a running container
type DockerSignal string

// Docker Signals
const (
	SigTerm = DockerSignal("TERM")
	SigChld = DockerSignal("CHLD")
	SigUsr1 = DockerSignal("USR1")
	SigHup  = DockerSignal("HUP")
)

// DockerProbe is a test probe for docker
type DockerProbe interface {
	SendSignal(containerID string, signal DockerSignal) error
}

type dockerClient struct {
	Client docker.Client
}

// NewDockerProbe creates a new DockerProbe for testing docker
func NewDockerProbe() (DockerProbe, error) {
	client, err := docker.NewDockerClient(dockerSocket, nil)
	if err != nil {
		return nil, err
	}
	return DockerProbe(dockerClient{Client: client}), nil
}

func (c dockerClient) SendSignal(containerID string, signal DockerSignal) error {
	return c.Client.KillContainer(containerID, string(signal))
}
