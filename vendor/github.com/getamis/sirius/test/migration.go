package test

import (
	"fmt"

	"github.com/getamis/sirius/log"
)

// MigrationOptions for sql migration container
type MigrationOptions struct {
	ImageRepository string
	ImageTag        string

	// this command will override the default command.
	// "bundle" "exec" "rake" "db:migrate"
	Command []string
}

// RunMigrationContainer creates the migration container and connects to the
// sql database container to run the migration scripts.
func RunMigrationContainer(dbContainer *SQLContainer, options MigrationOptions) error {
	// the default command
	command := []string{"bundle", "exec", "rake", "db:migrate"}
	if len(options.Command) > 0 {
		command = options.Command
	}

	if len(options.ImageTag) == 0 {
		options.ImageTag = "latest"
	}

	// host = 127.0.0.1 means we run a sql server on host,
	// however the migration container needs to connect to the host from the container.
	// so that we need to override the host name
	// please note that is only supported on OS X
	//
	// when sql.Container is defined, which means we've created the
	// sql container in the runtime, we need to inspect the address of the docker container.
	if dbContainer.Options.Host == "127.0.0.1" {
		dbContainer.Options.Host = "host.docker.internal"
	} else if dbContainer.Container != nil {
		inspectedContainer, err := dbContainer.Container.dockerClient.InspectContainer(dbContainer.Container.container.ID)
		if err != nil {
			return err
		}

		// Override the sql host because the migration needs to connect to the
		// sql server via the docker bridge network directly.
		dbContainer.Options.Host = inspectedContainer.NetworkSettings.IPAddress
		for k := range inspectedContainer.Config.ExposedPorts {
			dbContainer.Options.Port = k.Port()
			break
		}
	}

	container := NewDockerContainer(
		ImageRepository(options.ImageRepository),
		ImageTag(options.ImageTag),
		DockerEnv(
			[]string{
				"RAILS_ENV=customized",
				fmt.Sprintf("HOST=%s", dbContainer.Options.Host),
				fmt.Sprintf("PORT=%s", dbContainer.Options.Port),
				fmt.Sprintf("DATABASE=%s", dbContainer.Options.Database),
				fmt.Sprintf("USERNAME=%s", dbContainer.Options.Username),
				fmt.Sprintf("PASSWORD=%s", dbContainer.Options.Password),
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

// RunGoMigrationContainer creates the migration container and connects to the
// sql database container to run the migration scripts.
func RunGoMigrationContainer(dbContainer *SQLContainer, options MigrationOptions) error {
	if len(options.ImageTag) == 0 {
		options.ImageTag = "latest"
	}

	// host = 127.0.0.1 means we run a sql server on host,
	// however the migration container needs to connect to the host from the container.
	// so that we need to override the host name
	// please note that is only supported on OS X
	//
	// when sql.Container is defined, which means we've created the
	// sql container in the runtime, we need to inspect the address of the docker container.
	if dbContainer.Options.Host == "127.0.0.1" {
		dbContainer.Options.Host = "host.docker.internal"
	} else if dbContainer.Container != nil {
		inspectedContainer, err := dbContainer.Container.dockerClient.InspectContainer(dbContainer.Container.container.ID)
		if err != nil {
			return err
		}

		// Override the sql host because the migration needs to connect to the
		// sql server via the docker bridge network directly.
		dbContainer.Options.Host = inspectedContainer.NetworkSettings.IPAddress
		for k := range inspectedContainer.Config.ExposedPorts {
			dbContainer.Options.Port = k.Port()
			break
		}
	}

	connectionString, err := dbContainer.Options.ToConnectionString()
	if err != nil {
		return err
	}
	dbString := fmt.Sprintf("%s://%s", dbContainer.Options.Driver, connectionString)
	// the default command
	command := []string{"-source", "file://migration", "-database", dbString, "-verbose", "up"}
	if len(options.Command) > 0 {
		command = options.Command
	}

	container := NewDockerContainer(
		ImageRepository(options.ImageRepository),
		ImageTag(options.ImageTag),
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
