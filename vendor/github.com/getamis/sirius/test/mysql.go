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
	"os"
	"time"

	"github.com/getamis/sirius/database/mysql"
	"github.com/getamis/sirius/log"
)

var DefaultMySQLOptions = SQLOptions{
	Driver:   "mysql",
	Username: "root",
	Password: "my-secret-pw",

	// port 3307 is used to be published on the host.
	// the port number will be changed to 3306 when we connect to the mysql container from
	// another container.
	Port: "3307",

	// The db we want to run the test
	Database: "db0",

	// the mysql host to be connected from the client
	// if we're running test on the host, we might need to connect to the mysql
	// server via 127.0.0.1:3307. however if we want to run the test inside the container,
	// we need to inspect the IP of the container
	// This field will be updated when using LoadSQLOptions
	Host: "",
}

type MySQLContainer struct {
	*SQLContainer
}

func (container *MySQLContainer) Start() error {
	err := container.Container.Start()
	if err != nil {
		return err
	}

	if err := container.Options.UpdateHostFromContainer(container.Container); err != nil {
		return err
	}

	connectionString, _ := container.Options.ToConnectionString()
	container.URL = connectionString
	return nil
}

func (container *MySQLContainer) Teardown() error {
	if container.Container != nil && container.Container.Started {
		return container.Container.Stop()
	}

	db, err := sql.Open("mysql", container.URL)
	if err != nil {
		return err
	}
	defer db.Close()

	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", container.Options.Database)
	if _, err = db.Exec(sql); err != nil {
		return err
	}

	return nil
}

func NewMySQLHealthChecker(options SQLOptions) ContainerCallback {
	return func(c *Container) error {
		// We use this connection string to verify the mysql container is ready.
		if err := options.UpdateHostFromContainer(c); err != nil {
			return err
		}
		connectionString, err := options.ToConnectionString()
		if err != nil {
			return err
		}

		return retry(10, 5*time.Second, func() error {
			log.Debug("Checking mysql status", "conn", connectionString)
			db, err := sql.Open("mysql", connectionString)
			if err != nil {
				return err
			}
			defer db.Close()
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", options.Database))
			return err
		})
	}
}

// LoadMySQLOptions returns the mysql options that will be used for the test
// cases to connect to.
func LoadMySQLOptions() SQLOptions {
	options := DefaultMySQLOptions

	// mysql container exposes port at 3306, if we're inside a container, we
	// need to use 3306 to connect to the mysql server.
	if IsInsideContainer() {
		options.Port = "3306"
	} else {
		options.Host = "127.0.0.1"
	}

	if host, ok := os.LookupEnv("TEST_MYSQL_HOST"); ok {
		options.Host = host
	}
	if val, ok := os.LookupEnv("TEST_MYSQL_PORT"); ok {
		options.Port = val
	}

	if val, ok := os.LookupEnv("TEST_MYSQL_DATABASE"); ok {
		options.Database = val
	}

	if val, ok := os.LookupEnv("TEST_MYSQL_USERNAME"); ok {
		options.Username = val
	}

	if val, ok := os.LookupEnv("TEST_MYSQL_PASSWORD"); ok {
		options.Password = val
	}
	return options
}

func createMySQLDatabase(options SQLOptions) error {
	// We must pass mysql.Database to the connection string function, if we
	// don't, the connection string will use "db" as the default database.
	// see https://maicoin.slack.com/archives/G0PKWFTNY/p1539335776000100 for more details.
	connectionString, err := mysql.ToConnectionString(
		mysql.Connector(mysql.DefaultProtocol, options.Host, options.Port),
		mysql.Database(""),
		mysql.UserInfo(options.Username, options.Password),
	)
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", options.Database)
	_, err = db.Exec(sql)
	return err
}

// setup the mysql connection
// if TEST_MYSQL_HOST is defined, then we will use the connection directly.
// if not, a mysql container will be started
func SetupMySQL() (*MySQLContainer, error) {
	options := LoadMySQLOptions()
	if _, ok := os.LookupEnv("TEST_MYSQL_HOST"); ok {

		connectionString, err := mysql.ToConnectionString(
			mysql.Connector(mysql.DefaultProtocol, options.Host, options.Port),
			mysql.Database(options.Database),
			mysql.UserInfo(options.Username, options.Password),
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to create mysql connection string: %v", err)
		}

		if err := createMySQLDatabase(options); err != nil {
			return nil, fmt.Errorf("Failed to create mysql database: %v", err)
		}

		return &MySQLContainer{
			SQLContainer: &SQLContainer{
				Options: options,
				URL:     connectionString,
			},
		}, nil
	}

	container, err := NewMySQLContainer(options)
	if err != nil {
		return nil, err
	}

	if err := container.Start(); err != nil {
		return container, err
	}

	return container, nil
}

func NewMySQLContainer(options SQLOptions, containerOptions ...Option) (*MySQLContainer, error) {
	// Once the mysql container is ready, we will create the database if it does not exist.
	checker := NewMySQLHealthChecker(options)

	// In order to let the tests connect to the mysql server, we need to
	// publish the container port 3306 to the host port 3307 only when we're on the host
	if IsInsideContainer() {
		containerOptions = append(containerOptions, ExposePorts("3306"))
	} else {
		// mysql container port always expose the server port on 3306
		containerOptions = append(containerOptions, ExposePorts("3306"))
		containerOptions = append(containerOptions, HostPortBindings(PortBinding{"3306/tcp", options.Port}))
	}

	// Create the container, please note that the container is not started yet.
	container := &MySQLContainer{
		SQLContainer: &SQLContainer{
			Options: options,
			Container: NewDockerContainer(
				// this is to keep some flexibility for passing extra container options..
				// however if we literally use "..." in the method call, an error
				// "too many arguments" will raise.
				append([]Option{
					ImageRepository("mysql"),
					ImageTag("5.7"),
					DockerEnv(
						[]string{
							fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", options.Password),
							fmt.Sprintf("MYSQL_DATABASE=%s", options.Database),
						},
					),
					HealthChecker(checker),
				}, containerOptions...)...,
			),
		},
	}

	// please note that: in order to get the correct container address, the
	// connection string will be updated when the container is started.
	connectionString, _ := options.ToConnectionString()
	container.URL = connectionString
	return container, nil
}
