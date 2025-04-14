package masterconn

import (
	"testing_system/common/config"
	"testing_system/common/connectors/invokerconn"
)

type Connector struct {
	// TODO: Add master connection
}

func NewConnector(connection *config.Connection) *Connector {
	return nil
}

func (c *Connector) InvokerJobResult(result *InvokerJobResult) error {
	return nil
}

func (c *Connector) SendInvokerStatus(response *invokerconn.StatusResponse) error {
	return nil
}
