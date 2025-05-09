package tsapi

import (
	"testing_system/clients/common"
	"testing_system/clients/tsapi/masterstatus"
	"testing_system/clients/tsapi/tsapiconfig"
	"testing_system/common/constants/resource"
	"testing_system/lib/logger"
)

type Handler struct {
	base         *common.ClientBase
	config       *tsapiconfig.Config
	masterStatus *masterstatus.MasterStatus
}

func SetupHandler(clientBase *common.ClientBase) error {
	h := &Handler{
		base: clientBase,
	}
	if clientBase.Config.TestingSystemAPI == nil {
		return logger.Error("No admin config specified")
	}
	h.config = clientBase.Config.TestingSystemAPI
	if h.config.LoadFilesHead == 0 {
		h.config.LoadFilesHead = tsapiconfig.DefaultLoadFilesHead
	}

	var err error
	h.masterStatus, err = masterstatus.NewMasterStatus(h.base)
	if err != nil {
		return err
	}

	h.setupRoutes()

	return nil
}

func (h *Handler) setupRoutes() {
	apiRouter := h.base.Router.Group("/api", h.base.RequireAuthMiddleware(false))
	apiCSRFRouter := apiRouter.Group("", h.base.CSRFMiddleware)

	apiRouter.GET("/get/problems", h.getProblems)
	apiRouter.GET("/get/problem/:id", h.getProblem)
	apiRouter.GET("/get/problem/:id/test/:test/input", h.problemTestResourceGetter(resource.TestInput))
	apiRouter.GET("/get/problem/:id/test/:test/answer", h.problemTestResourceGetter(resource.TestAnswer))

	apiCSRFRouter.PUT("/new/problem", h.addProblem)
	apiCSRFRouter.POST("/modify/problem/:id", h.modifyProblem)

	apiRouter.GET("/get/submissions", h.getSubmissions)
	apiRouter.GET("/get/submission/:id", h.getSubmission)
	apiRouter.GET("/get/submission/:id/source", h.submissionResourceGetter(resource.SourceCode, false))
	apiRouter.GET("/get/submission/:id/compile_output", h.submissionResourceGetter(resource.CompileOutput, true))

	apiRouter.GET("/get/submission/:id/test/:test/output", h.submissionTestResourceGetter(resource.TestOutput))
	apiRouter.GET("/get/submission/:id/test/:test/stderr", h.submissionTestResourceGetter(resource.TestStderr))
	apiRouter.GET("/get/submission/:id/test/:test/check", h.submissionTestResourceGetter(resource.CheckerOutput))

	apiCSRFRouter.PUT("/new/submission", h.addSubmission)

	apiRouter.GET("/get/master_status", h.getMasterStatus)

	apiCSRFRouter.POST("/reset/invoker_cache", h.resetInvokerCache)
}
