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
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/getamis/sirius/crypto/rand"
	"github.com/getamis/sirius/log"
)

type Container struct {
	dockerClient     *docker.Client
	name             string
	imageRespository string
	imageTag         string
	ports            []string
	runArgs          []string
	envs             []string
	container        *docker.Container
	healthChecker    healthChecker
	initializer      func(*Container) error
}

type healthChecker func(*Container) error

func NewDockerContainer(opts ...Option) *Container {
	c := &Container{
		dockerClient: newDockerClient(),
		healthChecker: func(c *Container) error {
			return nil
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	portBindings := make(map[docker.Port][]docker.PortBinding)
	exposedPorts := make(map[docker.Port]struct{})

	if len(c.ports) != 0 {
		for _, port := range c.ports {
			portBindings[docker.Port(port)] = []docker.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			}
			exposedPorts[docker.Port(port)] = struct{}{}
		}
	}

	var err error
	c.container, err = c.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: c.name + generateNameSuffix(),
		Config: &docker.Config{
			Image:        c.imageRespository + ":" + c.imageTag,
			Cmd:          c.runArgs,
			ExposedPorts: exposedPorts,
			Env:          c.envs,
		},
		HostConfig: &docker.HostConfig{
			PortBindings: portBindings,
		},
		Context: context.TODO(),
	})
	if err != nil {
		log.Error("Failed to create a container", "repository", c.imageRespository, "tag", c.imageTag, "err", err)
		return nil
	}

	return c
}

func newDockerClient() *docker.Client {
	var client *docker.Client
	if os.Getenv("DOCKER_MACHINE_NAME") != "" {
		client, _ = docker.NewClientFromEnv()
	} else {
		client, _ = docker.NewClient("unix:///var/run/docker.sock")
	}
	return client
}

func (c *Container) Start() error {
	err := c.dockerClient.StartContainer(c.container.ID, nil)
	if err != nil {
		return err
	}

	defer func() {
		if c.initializer != nil {
			err = c.initializer(c)
		}
	}()
	err = c.healthChecker(c)
	return err
}

func (c *Container) Suspend() error {
	return c.dockerClient.StopContainer(c.container.ID, 0)
}

func (c *Container) Wait() error {
	_, err := c.dockerClient.WaitContainer(c.container.ID)
	return err
}

func (c *Container) Stop() error {
	return c.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID:      c.container.ID,
		Force:   true,
		Context: context.TODO(),
	})
}

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
