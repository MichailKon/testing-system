package connectors

import (
	"github.com/go-resty/resty/v2"
	"testing_system/common/config"
)

type ConnectorBase struct {
	Connection *config.Connection
	client     *resty.Client
}

func NewConnectorBase(connection *config.Connection) *ConnectorBase {
	c := &ConnectorBase{
		Connection: connection,
		client:     resty.New(),
	}
	// TODO: Add host initialization from connection
	// TODO: Add auth
	// TODO: Add retry configuration
	return c
}

func (c *ConnectorBase) R() *resty.Request {
	return c.client.R()
}
