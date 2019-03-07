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

	"github.com/getamis/sirius/database/postgresql"
	"github.com/getamis/sirius/log"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	// ErrDuplicateDatabase returns if try to create duplicate database
	ErrDuplicateDatabase = "42P04"
)

var DefaultPostgreSQLOptions = SQLOptions{
	Driver:   "postgres",
	Username: "admin",
	Password: "12345",

	// port 5433 is used to be published on the host.
	// the port number will be changed to 5432 when we connect to the postgresql container from
	// another container.
	Port: "5433",

	// The db we want to run the test
	Database: "postgres",

	// the postgres host to be connected from the client
	// if we're running test on the host, we might need to connect to the postgres
	// server via 127.0.0.1:5433. however if we want to run the test inside the container,
	// we need to inspect the IP of the container
	// This field will be updated when using LoadPostgreSQLOptions
	Host: "",
}

type PostgreSQLContainer struct {
	*SQLContainer
	Args string
}

func (container *PostgreSQLContainer) Start() error {
	err := container.Container.Start()
	if err != nil {
		return err
	}

	if err := container.Options.UpdateHostFromContainer(container.Container); err != nil {
		return err
	}

	connectionString, _ := container.Options.ToConnectionString()
	container.URL = connectionString

	container.Args, _ = ToPostgresArgs(container.Options)
	return nil
}

func (container *PostgreSQLContainer) Teardown() error {
	if container.Container != nil && container.Container.Started {
		return container.Container.Stop()
	}

	db, err := sql.Open("postgres", container.Args)
	if err != nil {
		return err
	}
	defer db.Close()

	sql := fmt.Sprintf("DROP DATABASE %s", container.Options.Database)
	if _, err = db.Exec(sql); err != nil {
		return err
	}

	return nil
}

func NewPostgreSQLHealthChecker(options SQLOptions) ContainerCallback {
	return func(c *Container) error {
		// We use this connection string to verify the postgresql container is ready.
		if err := options.UpdateHostFromContainer(c); err != nil {
			return err
		}
		connectionArgs, err := ToPostgresArgs(options)
		if err != nil {
			return err
		}

		return retry(10, 5*time.Second, func() error {
			log.Debug("Checking postgresql status", "conn", connectionArgs)
			db, err := sql.Open("postgres", connectionArgs)
			if err != nil {
				return err
			}
			defer db.Close()

			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", options.Database))
			if IsDuplicateDatabase(err) {
				return nil
			}
			return err
		})
	}
}

func IsDuplicateDatabase(err error) bool {
	if err == nil {
		return false
	}
	if qpErr, ok := err.(*pq.Error); ok {
		if string(qpErr.Code) == ErrDuplicateDatabase {
			return true
		}
	}
	return false
}

// LoadPostgreSQLOptions returns the postgresql options that will be used for the test
// cases to connect to.
func LoadPostgreSQLOptions() SQLOptions {
	options := DefaultPostgreSQLOptions

	// postgresql container exposes port at 3306, if we're inside a container, we
	// need to use 3306 to connect to the postgresql server.
	if IsInsideContainer() {
		options.Port = "5432"
	} else {
		options.Host = "127.0.0.1"
	}

	if host, ok := os.LookupEnv("TEST_POSTGRESQL_HOST"); ok {
		options.Host = host
	}
	if val, ok := os.LookupEnv("TEST_POSTGRESQL_PORT"); ok {
		options.Port = val
	}

	if val, ok := os.LookupEnv("TEST_POSTGRESQL_DATABASE"); ok {
		options.Database = val
	}

	if val, ok := os.LookupEnv("TEST_POSTGRESQL_USERNAME"); ok {
		options.Username = val
	}

	if val, ok := os.LookupEnv("TEST_POSTGRESQL_PASSWORD"); ok {
		options.Password = val
	}
	return options
}

func createPostgreSQLDatabase(options SQLOptions) error {
	// We must pass postgresql.Database to the connection string function, if we
	// don't, the connection string will use "db" as the default database.
	// see https://maicoin.slack.com/archives/G0PKWFTNY/p1539335776000100 for more details.
	connectionArgs, err := postgresql.ToConnectionArgs(
		postgresql.Connector(options.Host, options.Port),
		postgresql.Database(""),
		postgresql.UserInfo(options.Username, options.Password),
	)
	if err != nil {
		return err
	}

	db, err := sql.Open("postgres", connectionArgs)
	if err != nil {
		return err
	}
	defer db.Close()

	sql := fmt.Sprintf("CREATE DATABASE %s", options.Database)
	_, err = db.Exec(sql)
	if IsDuplicateDatabase(err) {
		return nil
	}
	return err
}

// setup the postgresql connection
// if TEST_POSTGRESQL_HOST is defined, then we will use the connection directly.
// if not, a postgresql container will be started
func SetupPostgreSQL() (*PostgreSQLContainer, error) {
	options := LoadPostgreSQLOptions()
	if _, ok := os.LookupEnv("TEST_POSTGRESQL_HOST"); ok {

		connectionString, err := options.ToConnectionString()
		if err != nil {
			return nil, fmt.Errorf("Failed to create postgresql connection string: %v", err)
		}
		connectionargs, _ := ToPostgresArgs(options)

		if err := createPostgreSQLDatabase(options); err != nil {
			return nil, fmt.Errorf("Failed to create postgresql database: %v", err)
		}

		return &PostgreSQLContainer{
			SQLContainer: &SQLContainer{
				Options: options,
				URL:     connectionString,
			},
			Args: connectionargs,
		}, nil
	}

	container, err := NewPostgreSQLContainer(options)
	if err != nil {
		return nil, err
	}

	if err := container.Start(); err != nil {
		return container, err
	}

	return container, nil
}

func NewPostgreSQLContainer(options SQLOptions, containerOptions ...Option) (*PostgreSQLContainer, error) {
	// Once the postgresql container is ready, we will create the database if it does not exist.
	checker := NewPostgreSQLHealthChecker(options)

	// In order to let the tests connect to the postgresql server, we need to
	// publish the container port 3306 to the host port 5433 only when we're on the host
	if IsInsideContainer() {
		containerOptions = append(containerOptions, ExposePorts("5432"))
	} else {
		// postgresql container port always expose the server port on 3306
		containerOptions = append(containerOptions, ExposePorts("5432"))
		containerOptions = append(containerOptions, HostPortBindings(PortBinding{"5432/tcp", options.Port}))
	}

	// Create the container, please note that the container is not started yet.
	container := &PostgreSQLContainer{
		SQLContainer: &SQLContainer{
			Options: options,
			Container: NewDockerContainer(
				// this is to keep some flexibility for passing extra container options..
				// however if we literally use "..." in the method call, an error
				// "too many arguments" will raise.
				append([]Option{
					ImageRepository("postgres"),
					ImageTag("9.6"),
					DockerEnv(
						[]string{
							fmt.Sprintf("POSTGRES_USER=%s", options.Username),
							fmt.Sprintf("POSTGRES_PASSWORD=%s", options.Password),
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

	container.Args, _ = ToPostgresArgs(options)
	return container, nil
}

func ToPostgresArgs(o SQLOptions) (string, error) {
	return postgresql.ToConnectionArgs(
		postgresql.Connector(o.Host, o.Port),
		postgresql.Database(o.Database),
		postgresql.UserInfo(o.Username, o.Password),
	)
}
