package common

import "github.com/gin-gonic/gin"

func (b *ClientBase) RequireAuthMiddleware(redirect bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: add check auth
	}
}

func (b *ClientBase) CSRFMiddleware(c *gin.Context) {
	// TODO: Add check csrf
}
