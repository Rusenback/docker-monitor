package docker

import (
	"context"

	"github.com/docker/docker/client"
)

// Config sisältää Docker client konfiguraation
type Config struct {
	Host      string
	TLSVerify bool
	CertPath  string
}

// Client wrappaa Docker API clientin
type Client struct {
	cli *client.Client
	ctx context.Context
}

// NewClient luo uuden Docker clientin
func NewClient(cfg Config) (*Client, error) {
	// TODO: Implementoi myöhemmin
	return nil, nil
}

// Close sulkee yhteyden
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}
