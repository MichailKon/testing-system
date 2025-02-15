package masterconn

import "testing_system/common/config"

type Connector struct {
	// TODO: Add master connection
}

func NewConnector(connection *config.Connection) *Connector {
	return nil
}

func (c *Connector) InvokerJobResult(result *InvokerJobResult) error {
	return nil
}
