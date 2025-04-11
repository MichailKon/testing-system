package connector

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func RespOK[T any](c *gin.Context, data *T) {
	resp := &struct {
		OK   bool `json:"ok"`
		Data *T   `json:"data,omitempty"`
	}{
		OK:   true,
		Data: data,
	}

	c.JSON(http.StatusOK, resp)
}

func RespErr(c *gin.Context, code int, errf string, values ...interface{}) {
	c.JSON(code, &ErrResponse{
		OK:    false,
		Error: fmt.Sprintf(errf, values...),
	})
}
