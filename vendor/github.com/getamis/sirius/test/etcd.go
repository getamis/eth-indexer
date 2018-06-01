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
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Etcdcontainer struct {
	dockerContainer *Container
	URL             string
}

func (container *Etcdcontainer) Start() error {
	return container.dockerContainer.Start()
}

func (container *Etcdcontainer) Suspend() error {
	return container.dockerContainer.Suspend()
}

func (container *Etcdcontainer) Stop() error {
	return container.dockerContainer.Stop()
}

func NewEtcdContainer() (*Etcdcontainer, error) {
	port := 2379
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	checker := func(c *Container) error {
		return retry(10, 1*time.Second, func() error {
			resp, err := http.Get(fmt.Sprintf("%s/health", endpoint))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return errors.New(resp.Status)
			}
			return nil
		})
	}

	container := &Etcdcontainer{
		dockerContainer: NewDockerContainer(
			ImageRepository("quay.io/coreos/etcd"),
			ImageTag("v3.0.6"),
			Ports(port),
			RunOptions(
				[]string{
					"etcd", "-name", "etcd-test",
					"-advertise-client-urls", endpoint,
					"-listen-client-urls", fmt.Sprintf("http://0.0.0.0:%d", port),
				},
			),
			HealthChecker(checker),
		),
	}

	container.URL = fmt.Sprintf("http://127.0.0.1:%d", port)

	return container, nil
}
