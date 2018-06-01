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
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQContainer struct {
	dockerContainer *Container
	URL             string
}

func (container *RabbitMQContainer) Start() error {
	return container.dockerContainer.Start()
}

func (container *RabbitMQContainer) Suspend() error {
	return container.dockerContainer.Suspend()
}

func (container *RabbitMQContainer) Stop() error {
	return container.dockerContainer.Stop()
}

func NewRabbitMQContainer() (*RabbitMQContainer, error) {
	port := 5672
	guiPort := 15672
	endpoint := fmt.Sprintf("amqp://guest:guest@127.0.0.1:%d", port)
	checker := func(c *Container) error {
		return retry(10, 5*time.Second, func() error {
			conn, err := amqp.Dial(endpoint)
			defer conn.Close()
			return err
		})
	}
	container := &RabbitMQContainer{
		dockerContainer: NewDockerContainer(
			ImageRepository("rabbitmq"),
			ImageTag("3.6.2-management"),
			Ports(port, guiPort),
			HealthChecker(checker),
		),
	}

	container.URL = endpoint

	return container, nil
}
