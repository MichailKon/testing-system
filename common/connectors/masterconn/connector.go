package masterconn

import (
	"testing_system/common/config"
	"testing_system/common/connectors"
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/connector"

	"github.com/go-resty/resty/v2"
)

type Connector struct {
	connection *connectors.ConnectorBase
}

func NewConnector(connection *config.Connection) *Connector {
	return &Connector{connectors.NewConnectorBase(connection)}
}

func (c *Connector) SendInvokerJobResult(result *InvokerJobResult) error {
	r := c.connection.R()
	r.SetBody(result)

	return connector.ReceiveEmpty(r, "/master/invoker/job-result", resty.MethodPost)
}

func (c *Connector) SendInvokerStatus(response *invokerconn.Status) error {
	r := c.connection.R()
	r.SetBody(response)

	return connector.ReceiveEmpty(r, "/master/invoker/status", resty.MethodPost)
}
