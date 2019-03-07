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
	"os"

	"github.com/getamis/sirius/database/mysql"
	"github.com/getamis/sirius/database/postgresql"
)

type SQLOptions struct {
	Driver string

	// The following options are used in the connection string and the mysql server container itself.
	Username string
	Password string
	Port     string
	Database string

	// The host address that will be used to build the connection string
	Host string
}

// UpdateHostFromContainer updates the mysql host field according to the current environment
//
// If we're inside the container, we need to override the hostname
// defined in the option.
// If not, we should use the default value 127.0.0.1 because we will need to connect to the host port.
// please note that the TEST_MYSQL_HOST can be overridden.
func (o *SQLOptions) UpdateHostFromContainer(c *Container) error {
	if IsInsideContainer() {
		inspectedContainer, err := c.dockerClient.InspectContainer(c.container.ID)
		if err != nil {
			return err
		}
		o.Host = inspectedContainer.NetworkSettings.IPAddress
	}
	return nil
}

func (o *SQLOptions) ToConnectionString() (string, error) {
	switch o.Driver {
	case "mysql":
		return mysql.ToConnectionString(
			mysql.Connector(mysql.DefaultProtocol, o.Host, o.Port),
			mysql.Database(o.Database),
			mysql.UserInfo(o.Username, o.Password),
		)
	case "postgres":
		return postgresql.ToConnectionString(
			postgresql.Connector(o.Host, o.Port),
			postgresql.Database(o.Database),
			postgresql.UserInfo(o.Username, o.Password),
		)
	}

	return "", nil
}

type SQLContainer struct {
	*Container
	Options SQLOptions
	URL     string
}

func IsInsideContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/bin/running-in-container"); err == nil {
		return true
	}
	return false
}
