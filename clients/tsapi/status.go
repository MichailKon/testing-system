package tsapi

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) getMasterStatus(c *gin.Context) {
	respSuccess(c, h.masterStatus.Status())
}
