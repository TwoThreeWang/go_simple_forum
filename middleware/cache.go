package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kingwrcy/hn/utils"
)

type cacheWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *cacheWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// CacheMiddleware 缓存中间件
func CacheMiddleware(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只缓存GET请求
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		// 生成缓存key
		key := generateCacheKey(c)
		cache := c.MustGet("cache").(*utils.Cache)

		// 尝试获取缓存
		if data, exists := cache.Get(key); exists {
			c.Data(http.StatusOK, "text/html; charset=utf-8", data.([]byte))
			c.Abort()
			return
		}

		// 创建自定义ResponseWriter来捕获响应
		writer := &cacheWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// 处理请求
		c.Next()

		// 如果是成功的响应，则缓存
		if c.Writer.Status() == http.StatusOK {
			cache.Set(key, writer.body.Bytes(), duration)
		}
	}
}

// generateCacheKey 生成缓存key
func generateCacheKey(c *gin.Context) string {
	// 使用完整URL作为key的一部分
	data := c.Request.URL.String()

	// 计算hash作为缓存key
	hash := sha256.Sum256([]byte(data))
	return "route_cache:" + hex.EncodeToString(hash[:])
}
