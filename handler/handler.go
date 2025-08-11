package handler

import (
	"math/rand"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/kingwrcy/hn/middleware"
	"github.com/kingwrcy/hn/model"
	"github.com/kingwrcy/hn/provider"
	"github.com/kingwrcy/hn/utils"
	"github.com/kingwrcy/hn/vo"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func SetupRouter(injector *do.Injector, engine *gin.Engine) {
	provideHandlers(injector)

	userHandler := do.MustInvoke[*UserHandler](injector)
	indexHandler := do.MustInvoke[*IndexHandler](injector)
	postHandler := do.MustInvoke[*PostHandler](injector)
	inspectHandler := do.MustInvoke[*InspectHandler](injector)
	statisticsHandler := do.MustInvoke[*StatisticsHandler](injector)
	_ = do.MustInvoke[*CommentHandler](injector)
	cacheAdminHandler := do.MustInvoke[*CacheAdminHandler](injector) // 新增加
	// 静态文件
	engine.StaticFile("/ads.txt", "./static/ads.txt")       // ads.txt
	engine.StaticFile("/robots.txt", "./static/robots.txt") // robots.txt
	engine.StaticFile("/jump", "./templates/jump.html")     // 跳转页

	engine.GET("/settings", indexHandler.ToSettings)    // 系统设置
	engine.POST("/settings", indexHandler.SaveSettings) // 系统设置操作类
	engine.POST("/upload_img", indexHandler.UploadImg)  // 头像上传接口
	engine.GET("/hit", statisticsHandler.Hit)           // 统计信息收集
	engine.GET("/statistics", statisticsHandler.Query)  // 统计页
	engine.GET("/img_dl", utils.GetImg)                 // 图片代理

	engine.GET("/", middleware.CacheMiddleware(5*time.Minute), indexHandler.Index)                       // 热门帖子列表
	engine.GET("/sitemap.xml", indexHandler.SiteMap)                                                     // sitemap文件
	engine.GET("/feed", indexHandler.Feed)                                                               // rss文件
	engine.GET("/history", middleware.CacheMiddleware(5*time.Minute), indexHandler.History)              // 全部帖子列表
	engine.GET("/search", indexHandler.ToSearch)                                                         // 搜索页
	engine.GET("/new", indexHandler.ToNew)                                                               // 发布新贴
	engine.GET("/s/:pid", indexHandler.ToPost)                                                           //
	engine.GET("/resetPwd", indexHandler.ToResetPwd)                                                     // 重置密码申请页
	engine.POST("/resetPwd", indexHandler.DoResetPwd)                                                    // 重置密码操作类（发送重置链接邮件）
	engine.GET("/resetPwdEdit", indexHandler.ToResetPwdEdit)                                             // 重置密码操作页
	engine.POST("/resetPwdEdit", indexHandler.DoResetPwdEdit)                                            // 重置密码操作类
	engine.GET("/tags", middleware.CacheMiddleware(5*time.Minute), indexHandler.ToTags)                  // 标签页面
	engine.GET("/tags/edit/:id", indexHandler.ToEditTag)                                                 // 编辑标签页面
	engine.POST("/tags/edit", indexHandler.SaveTag)                                                      // 编辑标签操作类
	engine.GET("/tags/add", indexHandler.ToAddTag)                                                       // 新增标签
	engine.POST("/tags/remove", indexHandler.RemoveTag)                                                  // 删除标签
	engine.GET("/wait", indexHandler.ToWaitApproved)                                                     // 等待审核列表
	engine.GET("/comments", middleware.CacheMiddleware(5*time.Minute), indexHandler.ToComments)          // 全部评论列表
	engine.GET("/vote", indexHandler.Vote)                                                               // 投票
	engine.GET("/delcomment", indexHandler.DelComment)                                                   // 删除评论
	engine.GET("/moderations", middleware.CacheMiddleware(5*time.Minute), indexHandler.Moderation)       // 审核日志
	engine.GET("/d/:domainName", middleware.CacheMiddleware(5*time.Minute), indexHandler.SearchByDomain) // 根据分享域名获取帖子列表
	engine.POST("/search", indexHandler.DoSearch)                                                        // 搜索操作类
	engine.GET("/invite/:code", userHandler.ToInvited)                                                   // 邀请注册
	engine.POST("/invite/:code", userHandler.DoInvited)                                                  // 邀请注册操作类
	engine.GET("/type/:type", middleware.CacheMiddleware(5*time.Minute), postHandler.SearchByType)       // 根据类型获取帖子列表
	engine.GET("/users", middleware.CacheMiddleware(5*time.Minute), userHandler.ToList)                  // 用户列表
	engine.GET("/activate", indexHandler.Activate)                                                       // 发送激活邮件
	engine.POST("/inspect", inspectHandler.Inspect)                                                      // 帖子审核

	userGroup := engine.Group("/u")
	userGroup.POST("/login", userHandler.Login)                                                                 // 登录操作类
	userGroup.GET("/login", userHandler.ToLogin)                                                                // 登录
	userGroup.GET("/logout", userHandler.Logout)                                                                // 退出登录
	userGroup.GET("/profile/:userid", middleware.CacheMiddleware(5*time.Minute), userHandler.Links)             // 用户主页
	userGroup.GET("/profile/:userid/edit", userHandler.UserEdit)                                                // 用户信息修改
	userGroup.POST("/profile/edit", userHandler.SaveUser)                                                       // 用户信息修改操作类
	userGroup.GET("/profile/:userid/asks", middleware.CacheMiddleware(5*time.Minute), userHandler.Asks)         // 用户讨论贴子列表
	userGroup.GET("/profile/:userid/links", middleware.CacheMiddleware(5*time.Minute), userHandler.Links)       // 用户分享帖子列表
	userGroup.GET("/profile/:userid/comments", middleware.CacheMiddleware(5*time.Minute), userHandler.Comments) // 用户评论帖子列表
	userGroup.GET("/profile/:userid/collects", middleware.CacheMiddleware(5*time.Minute), userHandler.Collects) // 用户收藏帖子列表
	userGroup.GET("/message/setAllRead", userHandler.SetAllRead)                                                // 消息全部已读
	userGroup.GET("/message/setSingleRead", userHandler.SetSingleRead)                                          // 消息已读
	userGroup.GET("/message", userHandler.ToMessage)                                                            // 消息列表页
	userGroup.GET("/invite", userHandler.InviteList)                                                            // 邀请页
	userGroup.GET("/addinvite", userHandler.InviteNew)                                                          // 邀请码生成
	userGroup.GET("/status", userHandler.SetStatus)                                                             // 修改用户状态（激活或者禁止）
	userGroup.GET("/punch", userHandler.Punch)                                                                  // 签到

	engine.POST("/oauth/callback/google", userHandler.Oauth) // 三方登录

	//commentGroup := engine.Group("/c")
	//commentGroup.GET("/vote", commentHandler.Vote)

	postGroup := engine.Group("/p")
	postGroup.POST("/new", postHandler.Add)             // 发布新帖操作类
	postGroup.GET("/:pid", postHandler.Detail)          // 帖子详情
	postGroup.GET("/:pid/edit", postHandler.ToEdit)     // 帖子编辑
	postGroup.POST("/:pid/edit", postHandler.DoUpdate)  // 帖子编辑操作类
	postGroup.POST("/comment", postHandler.AddComment)  // 发布评论
	postGroup.GET("/click/:pid", postHandler.ClickPost) // 增加点击量

	tagGroup := engine.Group("/t")
	tagGroup.GET("/:tag", middleware.CacheMiddleware(5*time.Minute), postHandler.SearchByTag)         // 标签页
	tagGroup.GET("/p/:tag", middleware.CacheMiddleware(5*time.Minute), postHandler.SearchByParentTag) // 标签下帖子

	// 添加缓存管理路由（放到最后）
	engine.GET("/cache/clear", cacheAdminHandler.ClearCacheHandler)
}

// 在 provideHandlers 函数中添加
func provideHandlers(injector *do.Injector) {
	do.Provide(injector, NewIndexHandler)
	do.Provide(injector, NewUserHandler)
	do.Provide(injector, NewPostHandler)
	do.Provide(injector, newInspectHandler)
	do.Provide(injector, newCommentHandler)
	do.Provide(injector, NewStatisticsHandler)
	do.Provide(injector, NewCacheAdminHandler) // 新增加
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytesMaskImpr(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

func GetCurrentUser(c *gin.Context) *vo.Userinfo {
	session := sessions.Default(c)
	login := session.Get("login")
	if login != nil {
		userinfo := session.Get("userinfo")
		if v, ok := userinfo.(vo.Userinfo); ok {
			return &v
		}
	}
	return nil
}

func OutputCommonSession(injector *do.Injector, c *gin.Context, h ...gin.H) gin.H {
	session := sessions.Default(c)
	result := gin.H{}
	start := c.GetInt64("executionTime")
	db := do.MustInvoke[*gorm.DB](injector)
	config := do.MustInvoke[*provider.AppConfig](injector)

	result["login"] = session.Get("login")
	result["userinfo"] = session.Get("userinfo")
	for _, v := range h {
		for k1, v1 := range v {
			result[k1] = v1
		}
	}
	var total int64
	userinfo := GetCurrentUser(c)
	if userinfo != nil {
		db.Model(&model.TbMessage{}).Where("to_user_id = ? and read = 'N'", userinfo.ID).Count(&total)
		result["unReadMessageCount"] = total
	}
	if userinfo != nil && (userinfo.Role == "admin" || userinfo.Role == "inspector") {
		db.Model(&model.TbPost{}).Where("status = 'Wait'").Count(&total)
		result["waitApproved"] = total
	}
	var settings model.TbSettings
	db.First(&settings)

	result["siteName"] = os.Getenv("SiteName")
	result["ClientID"] = os.Getenv("ClientID")
	result["SiteUrl"] = os.Getenv("SiteUrl")
	result["path"] = c.Request.URL.Path
	result["refer"] = c.Request.Referer()
	result["VERSION"] = os.Getenv("VERSION")
	result["cacheTime"] = time.Now().Format("15:04:05")
	result["settings"] = settings.Content
	result["staticCdnPrefix"] = config.StaticCdnPrefix
	result["executionTime"] = time.Since(time.UnixMilli(start)).Milliseconds()
	return result
}
