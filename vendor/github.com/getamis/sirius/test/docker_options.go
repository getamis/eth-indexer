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

import "fmt"

type Option func(*Container)

func Name(name string) Option {
	return func(c *Container) {
		c.name = name
	}
}

func ImageRepository(name string) Option {
	return func(c *Container) {
		c.imageRespository = name
	}
}

func ImageTag(tag string) Option {
	return func(c *Container) {
		c.imageTag = tag
	}
}

type PortBinding struct {
	ContainerPort string
	HostPort      string
}

func HostPortBindings(bindings ...PortBinding) Option {
	return func(c *Container) {
		for _, binding := range bindings {
			c.addHostPortBinding(binding.ContainerPort, binding.HostPort)
		}
	}
}

func ExposePorts(ports ...string) Option {
	return func(c *Container) {
		for _, port := range ports {
			c.exposePort(port)
		}
	}
}

// Ports function automatically combines port bindings and exposes ports of the
// container
func Ports(ports ...int) Option {
	return func(c *Container) {
		var p []string
		for _, port := range ports {
			p = append(p, fmt.Sprintf("%d", port))
		}
		c.ports = p
	}
}

func RunOptions(args []string) Option {
	return func(c *Container) {
		c.runArgs = args
	}
}

func DockerEnv(env []string) Option {
	return func(c *Container) {
		c.envs = env
	}
}

func HealthChecker(checker ContainerCallback) Option {
	return func(c *Container) {
		c.healthChecker = checker
	}
}

func Initializer(init ContainerCallback) Option {
	return func(c *Container) {
		c.initializer = init
	}
}
