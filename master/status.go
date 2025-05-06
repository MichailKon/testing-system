package master

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// @Summary Status
// @Description Status of master
// @Tags Client
// @Produce json
// @Param prevEpoch query string false "epoch of previous update" example:"long-epoch-name"
// @Success 200 {object} masterconn.Status
// @Failure 500 {object} string
// @Router /master/status [get]
func (m *Master) handleStatus(c *gin.Context) {
	prevEpoch := c.Query("prevEpoch")
	status := m.queue.Status().GetStatus(prevEpoch)
	status.Invokers = m.invokerRegistry.Status()
	c.JSON(http.StatusOK, status)
}
