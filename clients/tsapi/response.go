package tsapi

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xorcare/pointer"
	"net/http"
	"testing_system/lib/logger"
)

type response struct {
	Ok       bool        `json:"ok"`
	Error    *string     `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
}

func respServerError(c *gin.Context, format string, args ...interface{}) {
	logger.ErrorLevel(1, format, args...)
	c.JSON(http.StatusInternalServerError, response{
		Ok:    false,
		Error: pointer.String("internal error"),
	})
}

func respError(c *gin.Context, code int, format string, values ...interface{}) {
	c.JSON(code, response{
		Ok:    false,
		Error: pointer.String(fmt.Sprintf(format, values...)),
	})
}

func respSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response{
		Ok:       true,
		Response: data,
	})
}

func respSuccessEmpty(c *gin.Context) {
	c.JSON(http.StatusOK, response{
		Ok: true,
	})
}
