package middleware

import (
	"github.com/gin-gonic/gin"
	"time"
)

func CostHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("executionTime", time.Now().UnixMilli())
		c.Set("Cache-Control", "public, max-age=7200")
		c.Next()
	}

}
