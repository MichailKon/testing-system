package common

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"testing_system/lib/logger"

	swaggo "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

func (ts *TestingSystem) recoverRequest(c *gin.Context, err any) {
	if err != nil {
		ts.panicsLock.Lock()
		defer ts.panicsLock.Unlock()
		ts.panics = append(ts.panics, err)

		ts.stopFunc()
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (ts *TestingSystem) InitServer() {
	gin.SetMode(gin.ReleaseMode)
	ts.Router = gin.New()

	if logger.GetLevel() <= logger.LogLevelTrace {
		ts.Router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			Output: logger.CreateWriter(logger.LogLevelTrace, "Handler log:"),
		}))
	}
	ts.Router.Use(gin.CustomRecoveryWithWriter(
		logger.CreateWriter(logger.LogLevelError, "Panic in handler:"),
		ts.recoverRequest,
	))

	ts.Router.GET("/swagger/*any", ginswagger.WrapHandler(swaggo.Handler))
}

func (ts *TestingSystem) runServer() {
	addr := ":" + strconv.Itoa(ts.Config.Port)
	if ts.Config.Host != nil {
		addr = *ts.Config.Host + addr
	}
	logger.Info("Starting server at " + addr)
	server := http.Server{
		Addr:    addr,
		Handler: ts.Router,
	}
	go func() {
		<-ts.StopCtx.Done()
		logger.Info("Shutting down server")
		server.Shutdown(context.Background())
	}()
	server.ListenAndServe()
}
