package invoker

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (i *Invoker) HandlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive":        true,
		"queueSize:":   len(i.Queue),
		"maxQueueSize": i.MaxQueueSize,
	})
}

func (i *Invoker) HandleTest(c *gin.Context) {
	// TODO
}
