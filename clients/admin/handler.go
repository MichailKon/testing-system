package admin

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"testing_system/clients/common"
	"testing_system/lib/logger"
)

type Handler struct {
	base *common.ClientBase
}

func SetupHandler(base *common.ClientBase) error {
	h := &Handler{
		base: base,
	}
	if base.Config.Admin == false {
		return logger.Error("No admin config specified")
	}
	h.setupRoutes()

	return nil
}

func (h *Handler) setupRoutes() {
	router := h.base.Router.Group("/admin", h.base.RequireAuthMiddleware(true))
	router.GET("", h.serveFrontend)

	router.GET("/problems", h.serveFrontend)
	router.GET("/new/problem", h.serveFrontend)
	router.GET("/problem/:id", h.serveFrontend)

	router.GET("/submissions", h.serveFrontend)
	router.GET("/submission/:id", h.serveFrontend)
	router.GET("/new/submission", h.serveFrontend)

	router.GET("/status", h.serveFrontend)
}

func (h *Handler) serveFrontend(c *gin.Context) {
	c.HTML(
		http.StatusOK,
		"admin.gohtml",
		gin.H{},
	)
}
