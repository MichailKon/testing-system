package tsapi

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"testing_system/lib/logger"
)

func (h *Handler) getMasterStatus(c *gin.Context) {
	status, err := h.masterStatus.Status()
	if err != nil {
		logger.Error("Get master status failed: %v", err)
		respError(c, http.StatusServiceUnavailable, "%v", err.Error())
		return
	}
	respSuccess(c, status)
}

func (h *Handler) resetInvokerCache(c *gin.Context) {
	err := h.base.MasterConnection.ResetInvokerCache(c)
	if err != nil {
		logger.Error("Reset invoker cache failed: %v", err)
		respError(c, http.StatusInternalServerError, "server error")
		return
	}
	respSuccess(c, nil)
}
