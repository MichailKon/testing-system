package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RespOK(c *gin.Context, data gin.H) {
	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"response": data,
	})
}

func RespErr(c *gin.Context, code int, errf string, values ...interface{}) {
	c.JSON(code, gin.H{
		"ok":    false,
		"error": fmt.Sprintf(errf, values...),
	})
}
