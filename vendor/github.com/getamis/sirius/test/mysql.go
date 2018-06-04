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
	"database/sql"
	"fmt"
	"time"

	"github.com/getamis/sirius/database/mysql"
	"github.com/getamis/sirius/log"
)

type MySQLContainer struct {
	dockerContainer *Container
	URL             string
}

func (container *MySQLContainer) Start() error {
	return container.dockerContainer.Start()
}

func (container *MySQLContainer) Suspend() error {
	return container.dockerContainer.Suspend()
}

func (container *MySQLContainer) Stop() error {
	return container.dockerContainer.Stop()
}

func NewMySQLContainer(migrationRepository string) (*MySQLContainer, error) {
	port := 3306
	password := "my-secret-pw"
	database := "db0"
	connectionString, _ := mysql.ToConnectionString(
		mysql.Connector(mysql.DefaultProtocol, "127.0.0.1", fmt.Sprintf("%d", port)),
		mysql.Database(database),
		mysql.UserInfo("root", password),
	)
	checker := func(c *Container) error {
		return retry(10, 5*time.Second, func() error {
			db, err := sql.Open("mysql", connectionString)
			if err != nil {
				return err
			}
			defer db.Close()
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database))
			return err
		})
	}
	container := &MySQLContainer{
		dockerContainer: NewDockerContainer(
			ImageRepository("mysql"),
			ImageTag("5.7"),
			Ports(port),
			DockerEnv(
				[]string{
					fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", password),
					fmt.Sprintf("MYSQL_DATABASE=%s", database),
				},
			),
			HealthChecker(checker),
			Initializer(func(c *Container) error {
				inspectedContainer, err := c.dockerClient.InspectContainer(c.container.ID)
				if err != nil {
					return err
				}
				if migrationRepository == "" {
					return nil
				}

				migrationContainer := NewDockerContainer(
					ImageRepository(migrationRepository),
					ImageTag("latest"),
					DockerEnv(
						[]string{
							"RAILS_ENV=customized",
							fmt.Sprintf("HOST=%s", inspectedContainer.NetworkSettings.IPAddress),
							fmt.Sprintf("PORT=%d", port),
							fmt.Sprintf("DATABASE=%s", database),
							"USERNAME=root",
							fmt.Sprintf("PASSWORD=%s", password),
						},
					),
					RunOptions(
						[]string{
							"bundle", "exec", "rake", "db:migrate",
						},
					),
				)

				if err := migrationContainer.Start(); err != nil {
					log.Error("Failed to start container", "err", err)
					return err
				}

				if err := migrationContainer.Wait(); err != nil {
					log.Error("Failed to wait container", "err", err)
					return err
				}

				return migrationContainer.Stop()
			}),
		),
	}

	container.URL = connectionString

	return container, nil
}
