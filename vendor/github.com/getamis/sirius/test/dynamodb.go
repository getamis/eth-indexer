package test

import (
	"net"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/getamis/sirius/log"
)

const DefaultDynamodbPort = "8000"

type DynamodbOptions struct {
	Host   string
	Port   string
	Region string
}

func (o DynamodbOptions) Endpoint() string {
	return "http://" + net.JoinHostPort(o.Host, o.Port)
}

func (o DynamodbOptions) MustNewSession() *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(o.Region),
		Endpoint:    aws.String(o.Endpoint()),
		Credentials: credentials.NewStaticCredentials("FAKE", "FAKE", "FAKE"),
	}))
}

// UpdateHostFromContainer updates the mysql host field according to the current environment
//
// If we're inside the container, we need to override the hostname
// defined in the option.
// If not, we should use the default value 127.0.0.1 because we will need to connect to the host port.
// please note that the TEST_MYSQL_HOST can be overridden.
func (o *DynamodbOptions) UpdateHostFromContainer(c *Container) error {
	if IsInsideContainer() {
		inspectedContainer, err := c.dockerClient.InspectContainer(c.container.ID)
		if err != nil {
			return err
		}
		o.Host = inspectedContainer.NetworkSettings.IPAddress
	}
	return nil
}

type DynamodbContainer struct {
	*Container
	Options  DynamodbOptions
	Endpoint string
}

func (c *DynamodbContainer) Start() error {
	err := c.Container.Start()
	if err != nil {
		return err
	}

	if err := c.Options.UpdateHostFromContainer(c.Container); err != nil {
		return err
	}

	c.Endpoint = c.Options.Endpoint()
	return nil
}

func (container *DynamodbContainer) Teardown() error {
	if container.Container != nil && container.Container.Started {
		return container.Container.Stop()
	}

	sess := container.Options.MustNewSession()
	svc := dynamodb.New(sess)

	input := &dynamodb.ListTablesInput{}
	result, err := svc.ListTables(input)
	if err != nil {
		return err
	}

	for _, n := range result.TableNames {
		log.Debug("Deleting dynamodb table", "table", *n)
		out, err := svc.DeleteTable(&dynamodb.DeleteTableInput{
			TableName: n,
		})
		if err != nil {
			log.Error("Failed to delete dynamodb table", "err", err)
		} else {
			log.Debug("Deleted", "output", out.String())
		}
	}

	return nil
}

// LoadDynamodbOptions returns the dynamodb options that will be used for the test
// cases to connect to.
func LoadDynamodbOptions() DynamodbOptions {
	options := DynamodbOptions{
		Host:   "localhost",
		Port:   DefaultDynamodbPort,
		Region: "ap-southeast-1",
	}
	if host, ok := os.LookupEnv("TEST_DYNAMODB_HOST"); ok {
		options.Host = host
	}
	if val, ok := os.LookupEnv("TEST_DYNAMODB_PORT"); ok {
		options.Port = val
	}
	if val, ok := os.LookupEnv("TEST_DYNAMODB_REGION"); ok {
		options.Region = val
	}
	return options
}

func SetupDynamodb() (*DynamodbContainer, error) {
	options := LoadDynamodbOptions()

	// Explicit dynamodb host is specified
	if _, ok := os.LookupEnv("TEST_DYNAMODB_HOST"); ok {
		return &DynamodbContainer{
			Options:  options,
			Endpoint: options.Endpoint(),
		}, nil
	}

	c, err := NewDynamodbContainer(options)

	if err := c.Start(); err != nil {
		return c, err
	}

	return c, err
}

func NewDynamodbHealthChecker(options DynamodbOptions) ContainerCallback {
	return func(c *Container) error {
		if IsInsideContainer() {
			if err := options.UpdateHostFromContainer(c); err != nil {
				return err
			}
		}

		return retry(10, 1*time.Second, func() error {
			log.Debug("Checking dynamodb status", "endpoint", options.Endpoint(), "region", options.Region)
			sess := options.MustNewSession()
			svc := dynamodb.New(sess)
			input := &dynamodb.ListTablesInput{}
			_, err := svc.ListTables(input)
			return err
		})
	}
}

func NewDynamodbContainer(options DynamodbOptions, containerOptions ...Option) (*DynamodbContainer, error) {
	// Once the mysql container is ready, we will create the database if it does not exist.
	checker := NewDynamodbHealthChecker(options)

	if IsInsideContainer() {
		containerOptions = append(containerOptions, ExposePorts(DefaultDynamodbPort))
	} else {
		// dynamodb container port always expose the server port on 8000
		// bind the dynamodb default port to the custom port on the host.
		containerOptions = append(containerOptions, ExposePorts(DefaultDynamodbPort))
		containerOptions = append(containerOptions, HostPortBindings(PortBinding{DefaultDynamodbPort + "/tcp", options.Port}))
	}

	// Create the container, please note that the container is not started yet.
	return &DynamodbContainer{
		Options: options,
		Container: NewDockerContainer(
			// this is to keep some flexibility for passing extra container options..
			// however if we literally use "..." in the method call, an error
			// "too many arguments" will raise.
			append([]Option{
				ImageRepository("amazon/dynamodb-local"),
				ImageTag("latest"),
				DockerEnv([]string{}),
				HealthChecker(checker),
			}, containerOptions...)...,
		),
		Endpoint: "http://" + net.JoinHostPort(options.Host, options.Port),
	}, nil
}
