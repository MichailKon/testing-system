package storageconn

import (
	"testing_system/common/config"
)

type Connector struct {
	// TODO: Add storage data loader
}

func NewConnector(connection *config.Connection) *Connector {
	return nil
}

func (s *Connector) Download(request *Request) *ResponseFiles {
	return nil
}

func (s *Connector) Upload(request *Request) *Response {
	return nil
}

func (s *Connector) Delete(request *Request) *Response {
	return nil
}
