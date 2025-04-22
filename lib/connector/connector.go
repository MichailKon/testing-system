package connector

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type Error struct {
	Code    int
	Message string
	Path    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("connector error, request path: %s, code: %d, message: %s", e.Path, e.Code, e.Message)
}

func Receive[T any](r *resty.Request, path string, method string) (*T, error) {
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
		Data  *T     `json:"data,omitempty"`
	}

	r.SetResult(&result)

	resp, err := r.Execute(method, path)
	if err != nil {
		return nil, err
	}
	if resp.IsError() || !result.OK {
		return nil, ParseRespError(resp.Body(), resp)
	}
	return result.Data, nil
}

func ReceiveEmpty(r *resty.Request, path string, method string) error {
	_, err := Receive[string](r, path, method)
	return err
}

func ParseRespError(body []byte, resp *resty.Response) error {
	var errResp ErrResponse
	err := json.Unmarshal(body, &errResp)
	if err != nil {
		return &Error{
			Code:    resp.StatusCode(),
			Message: fmt.Sprintf("failed to unmarshal response, error: %v", err),
			Path:    resp.Request.URL,
		}
	}

	return &Error{
		Code:    resp.StatusCode(),
		Message: errResp.Error,
		Path:    resp.Request.URL,
	}
}
