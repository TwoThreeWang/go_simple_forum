package handler

import (
	"go_simple_forum/middleware"
	"go_simple_forum/utils"
	"go_simple_forum/vo"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
)

type CacheAdminHandler struct {
	injector *do.Injector
}

func NewCacheAdminHandler(injector *do.Injector) (*CacheAdminHandler, error) {
	return &CacheAdminHandler{injector: injector}, nil
}

// ClearCacheHandler 清除缓存的接口
func (h *CacheAdminHandler) ClearCacheHandler(c *gin.Context) {
	// 验证管理员权限
	session := sessions.Default(c)
	userinfo := session.Get("userinfo")
	if userinfo == nil {
		c.JSON(403, gin.H{"error": "请先登录"})
		return
	}

	user := userinfo.(vo.Userinfo)
	if user.Role != "admin" {
		c.JSON(403, gin.H{"error": "无权限操作"})
		return
	}

	// 清除所有缓存
	cache := c.MustGet("cache").(*utils.Cache)
	clearedCount := middleware.ClearAllRouteCache(cache)

	c.JSON(200, gin.H{
		"message":      "缓存清除成功",
		"clearedCount": clearedCount,
	})
}
