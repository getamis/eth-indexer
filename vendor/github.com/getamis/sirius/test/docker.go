// Copyright 2017 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"context"
	"fmt"
	"os"

	"github.com/getamis/sirius/crypto/rand"
	"github.com/getamis/sirius/log"

	docker "github.com/fsouza/go-dockerclient"
)

type Container struct {
	dockerClient *docker.Client
	Started      bool

	name             string
	imageRespository string
	imageTag         string

	portBindings map[docker.Port][]docker.PortBinding
	exposedPorts map[docker.Port]struct{}

	ports         []string
	runArgs       []string
	envs          []string
	container     *docker.Container
	healthChecker ContainerCallback
	initializer   ContainerCallback
}

type ContainerCallback func(*Container) error

func newDockerClient() (*docker.Client, error) {
	if os.Getenv("DOCKER_MACHINE_NAME") != "" {
		return docker.NewClientFromEnv()
	}
	return docker.NewClient("unix:///var/run/docker.sock")
}

func NewDockerContainer(opts ...Option) *Container {
	client, err := newDockerClient()
	if err != nil {
		panic(err)
	}

	c := &Container{
		portBindings: make(map[docker.Port][]docker.PortBinding),
		exposedPorts: make(map[docker.Port]struct{}),
		dockerClient: client,
		healthChecker: func(c *Container) error {
			return nil
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Automatically convert the ports to exposed ports and host binding ports
	if len(c.ports) > 0 {
		for _, port := range c.ports {
			c.addHostPortBinding(port, port)
			c.exposePort(port)
		}
	}

	c.container, err = c.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: c.name + generateNameSuffix(),
		Config: &docker.Config{
			Image:        c.imageRespository + ":" + c.imageTag,
			Cmd:          c.runArgs,
			ExposedPorts: c.exposedPorts,
			Env:          c.envs,
		},
		HostConfig: &docker.HostConfig{
			PortBindings: c.portBindings,
		},
		Context: context.TODO(),
	})
	if err != nil {
		panic(fmt.Errorf("Failed to create a container %s:%s error:%s", c.imageRespository, c.imageTag, err))
	}

	return c
}

func (c *Container) OnReady(initializer ContainerCallback) {
	c.initializer = initializer
}

func (c *Container) Start() error {
	err := c.dockerClient.StartContainer(c.container.ID, nil)
	if err != nil {
		return err
	}

	defer log.Debug("Container IP address", "container ID", c.container.ID, "ip", c.IPAddress())

	c.Started = true
	defer func() {
		if c.initializer != nil {
			err = c.initializer(c)
		}
	}()
	err = c.healthChecker(c)
	return err
}

func (c *Container) exposePort(port string) {
	c.exposedPorts[docker.Port(port)] = struct{}{}
}

func (c *Container) addHostPortBinding(containerPort string, hostPort string) {
	c.portBindings[docker.Port(containerPort)] = []docker.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: hostPort,
		},
	}
}

func (c *Container) Suspend() error {
	defer func() {
		c.Started = false
	}()
	return c.dockerClient.StopContainer(c.container.ID, 0)
}

func (c *Container) Wait() error {
	_, err := c.dockerClient.WaitContainer(c.container.ID)
	return err
}

func (c *Container) Run() error {
	if err := c.Start(); err != nil {
		log.Error("Failed to start container", "err", err)
		return err
	}

	if err := c.Wait(); err != nil {
		log.Error("Failed to wait container", "err", err)
		return err
	}

	return c.Stop()
}

func (c *Container) Stop() error {
	defer func() {
		c.Started = false
	}()
	return c.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID:      c.container.ID,
		Force:   true,
		Context: context.TODO(),
	})
}

func (c *Container) SetHealthChecker(checker ContainerCallback) *Container {
	c.healthChecker = checker
	return c
}

func (c *Container) SetInitializer(initializer ContainerCallback) *Container {
	c.initializer = initializer
	return c
}

func (c *Container) SetEnvVars(envs []string) *Container {
	c.envs = envs
	return c
}

// IPAddress returns the IP adress of the container.
func (c *Container) IPAddress() string {
	spec, err := c.dockerClient.InspectContainer(c.container.ID)
	if err != nil {
		log.Error("Failed to get IPAddress", "err", err)
		return ""
	}
	return spec.NetworkSettings.IPAddress
}

// generateContainerID generates the UUID for container ID instead of the
// default name combinator
func generateContainerID() string {
	return rand.New(
		rand.HexEncoder(),
	).KeyEncoded()
}

func generateNameSuffix() string {
	return rand.New(
		rand.UUIDEncoder(),
	).KeyEncoded()
}
