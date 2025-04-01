package invokerconn

import (
	"github.com/go-resty/resty/v2"
	"testing_system/common/config"
	"testing_system/common/connectors"
	"testing_system/lib/connector"
)

type Connector struct {
	connection *connectors.ConnectorBase
}

func NewConnector(connection *config.Connection) *Connector {
	return &Connector{connectors.NewConnectorBase(connection)}
}

func (c *Connector) Status() (*StatusResponse, error) {
	r := c.connection.R()

	return connector.Receive[StatusResponse](r, "/invoker/status", resty.MethodGet)
}

func (c *Connector) NewJob(job *Job) (*StatusResponse, error) {
	r := c.connection.R()
	r.SetBody(job)

	return connector.Receive[StatusResponse](r, "/invoker/job/new", resty.MethodPost)
}
