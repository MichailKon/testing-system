package invokerconn

import (
	"testing_system/common/config"
	"testing_system/common/connectors"
	"testing_system/lib/connector"

	"github.com/go-resty/resty/v2"
)

type Connector struct {
	connection *connectors.ConnectorBase
}

func NewConnector(connection *config.Connection) *Connector {
	return &Connector{connectors.NewConnectorBase(connection)}
}

func (c *Connector) Status() (*Status, error) {
	r := c.connection.R()

	return connector.Receive[Status](r, "/invoker/status", resty.MethodGet)
}

func (c *Connector) NewJob(job *Job) (*Status, error) {
	r := c.connection.R()
	r.SetBody(job)

	return connector.Receive[Status](r, "/invoker/job/new", resty.MethodPost)
}

func (c *Connector) ResetCache() error {
	r := c.connection.R()
	return connector.ReceiveEmpty(r, "/invoker/reset_cache", resty.MethodPost)
}
