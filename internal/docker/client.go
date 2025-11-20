package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"
)

// Config contains Docker client configuration
type Config struct {
	Host      string
	TLSVerify bool
	CertPath  string
	Timeout   time.Duration
}

func DefaultConfig() Config {
	return Config{
		Host:    "unix:///var/run/docker.sock",
		Timeout: 30 * time.Second,
	}
}

// Client wraps the Docker API client
type Client struct {
	cli *client.Client
	Ctx context.Context
}

// NewClient creates a new Docker client
func NewClient(cfg Config) (*Client, error) {
	opts := []client.Opt{
		client.WithHost(cfg.Host),
		client.WithAPIVersionNegotiation(),
	}

	if cfg.TLSVerify {
		opts = append(opts, client.WithTLSClientConfig(
			cfg.CertPath+"/ca.pem",
			cfg.CertPath+"/cert.pem",
			cfg.CertPath+"/key.pem",
		))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: cli,
		Ctx: context.Background(),
	}, nil

}

// Close closes the connection
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}
