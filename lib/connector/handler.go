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

func RespOK(c *gin.Context, data any) {
	resp := &struct {
		OK   bool `json:"ok"`
		Data any  `json:"data,omitempty"`
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
