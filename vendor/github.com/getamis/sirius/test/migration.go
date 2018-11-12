package test

import (
	"fmt"

	"github.com/getamis/sirius/log"
)

// MigrationOptions for mysql migration container
type MigrationOptions struct {
	ImageRepository string
	ImageTag        string

	// this command will override the default command.
	// "bundle" "exec" "rake" "db:migrate"
	Command []string
}

// RunMigrationContainer creates the migration container and connects to the
// mysql database to run the migration scripts.
func RunMigrationContainer(mysql *MySQLContainer, options MigrationOptions) error {
	// the default command
	command := []string{"bundle", "exec", "rake", "db:migrate"}
	if len(options.Command) > 0 {
		command = options.Command
	}

	if len(options.ImageTag) == 0 {
		options.ImageTag = "latest"
	}

	// host = 127.0.0.1 means we run a mysql server on host,
	// however the migration container needs to connect to the host from the container.
	// so that we need to override the host name
	// please note that is only supported on OS X
	//
	// when mysql.Container is defined, which means we've created the
	// mysql container in the runtime, we need to inspect the address of the docker container.
	if mysql.Options.Host == "127.0.0.1" {
		mysql.Options.Host = "host.docker.internal"
	} else if mysql.Container != nil {
		inspectedContainer, err := mysql.Container.dockerClient.InspectContainer(mysql.Container.container.ID)
		if err != nil {
			return err
		}

		// Override the mysql host because the migration needs to connect to the
		// mysql server via the docker bridge network directly.
		mysql.Options.Host = inspectedContainer.NetworkSettings.IPAddress
		mysql.Options.Port = "3306"
	}

	container := NewDockerContainer(
		ImageRepository(options.ImageRepository),
		ImageTag(options.ImageTag),
		DockerEnv(
			[]string{
				"RAILS_ENV=customized",
				fmt.Sprintf("HOST=%s", mysql.Options.Host),
				fmt.Sprintf("PORT=%s", mysql.Options.Port),
				fmt.Sprintf("DATABASE=%s", mysql.Options.Database),
				fmt.Sprintf("USERNAME=%s", mysql.Options.Username),
				fmt.Sprintf("PASSWORD=%s", mysql.Options.Password),
			},
		),
		RunOptions(command),
	)

	if err := container.Start(); err != nil {
		log.Error("Failed to start container", "err", err)
		return err
	}

	if err := container.Wait(); err != nil {
		log.Error("Failed to wait container", "err", err)
		return err
	}

	return container.Stop()
}
