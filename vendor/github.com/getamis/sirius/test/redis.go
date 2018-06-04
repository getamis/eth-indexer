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

	redis "gopkg.in/redis.v5"
)

type RedisContainer struct {
	container *Container
	URL       string
}

func (container *RedisContainer) Start() error {
	return container.container.Start()
}

func (container *RedisContainer) Suspend() error {
	return container.container.Suspend()
}

func (container *RedisContainer) Stop() error {
	return container.container.Stop()
}

func NewRedisContainer() (*RedisContainer, error) {
	port := 6379
	endpoint := fmt.Sprintf("127.0.0.1:%d", port)
	checker := func(c *Container) error {
		return retry(10, 5*time.Second, func() error {
			c := redis.NewClient(&redis.Options{
				Addr: endpoint,
			})
			if c == nil {
				return fmt.Errorf("failed to connect to %s", endpoint)
			}
			return c.Ping().Err()
		})
	}
	container := &RedisContainer{
		container: NewDockerContainer(
			ImageRepository("redis"),
			ImageTag("3-alpine"),
			Ports(port),
			HealthChecker(checker),
		),
	}

	container.URL = endpoint

	return container, nil
}
