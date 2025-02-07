package storageconn

import (
	"testing_system/common"
)

type Connector struct {
	// TODO: Add storage data loader
}

func NewStorageConnector(ts *common.TestingSystem) *Connector {
	return nil
}

func (s *Connector) Download(request Request) *ResponseFiles {
	return nil
}

func (s *Connector) Upload(request Request) *Response {
	return nil
}

func (s *Connector) Delete(request Request) *Response {
	return nil
}
