package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/internal/registry/manager"
	"net/http"
)

// GlobalAPIRateLimit 全局API限流
func GlobalAPIRateLimit(c *gin.Context) {
	limiter := manager.GetManager().GetIPLimiter()
	cfg := manager.GetManager().GetConfig()
	if cfg.Limit.Enable && !limiter.GetLimiter(c.RemoteIP()).Allow() {
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
}
