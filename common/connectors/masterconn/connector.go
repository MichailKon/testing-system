package masterconn

import (
	"context"
	"io"
	"net/http"
	"strconv"
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
func (c *Connector) SendNewSubmission(
	ctx context.Context,
	problemID uint,
	language string,
	fileName string,
	fileReader io.Reader,
) (SubmissionID uint, err error) {
	r := c.connection.R()
	r.SetContext(ctx)
	r.SetFormData(map[string]string{
		"ProblemID": strconv.FormatUint(uint64(problemID), 10),
		"Language":  language,
	})
	r.SetFileReader("Solution", fileName, fileReader)
	var submissionResponse SubmissionResponse
	r.SetResult(&submissionResponse)
	resp, err := r.Post("/master/submit")
	if err != nil {
		return 0, err
	}
	if resp.StatusCode() != http.StatusOK {
		return 0, connector.ParseRespError(resp.Body(), resp)
	}
	return submissionResponse.SubmissionID, nil
}

func (c *Connector) GetStatus(ctx context.Context, prevEpoch string) (*Status, error) {
	r := c.connection.R()
	r.SetContext(ctx)
	r.SetQueryParam("prevEpoch", prevEpoch)
	var status Status
	r.SetResult(&status)
	resp, err := r.Get("/master/status")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, connector.ParseRespError(resp.Body(), resp)
	}
	return &status, nil
}

func (c *Connector) ResetInvokerCache(ctx context.Context) error {
	r := c.connection.R()
	r.SetContext(ctx)
	resp, err := r.Post("/master/reset_invoker_cache")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return connector.ParseRespError(resp.Body(), resp)
	}
	return nil
}
