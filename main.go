package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"time"

	"go_simple_forum/handler"
	"go_simple_forum/middleware"
	"go_simple_forum/model"
	"go_simple_forum/provider"
	"go_simple_forum/task"
	"go_simple_forum/utils"
	"go_simple_forum/vo"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/samber/do"

	"log"

	"gorm.io/gorm"
)

func main() {
	// 加载配置文件
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file" + err.Error())
	}
	// 创建依赖注入容器
	injector := do.New()
	// 注册两个提供者函数到注入器中，用于提供应用程序所需的配置信息和数据库实例
	do.Provide(injector, provider.NewAppConfig)
	do.Provide(injector, provider.NewRepository)
	// 从注入器中获取数据库连接实例和应用程序配置信息实例
	db := do.MustInvoke[*gorm.DB](injector)
	config := do.MustInvoke[*provider.AppConfig](injector)
	// 数据库初始化
	err = db.AutoMigrate(&model.TbMessage{},
		&model.TbUser{}, &model.TbInviteRecord{},
		&model.TbPost{}, &model.TbInspectLog{},
		&model.TbComment{}, &model.TbTag{}, &model.TbStatistics{},
		&model.TbVote{}, &model.TbSettings{})
	if err != nil {
		log.Fatalf("升级数据库异常,启动失败.%s", err)
		return
	}
	// 初始化数据库配置
	initSystem(db)
	// 注册自定义类型，为了在 Gin 的会话(session)中存储和读取 vo.Userinfo 类型的数据。
	gob.Register(vo.Userinfo{})

	// 设置gin运行模式
	gin.SetMode(config.GinMode)
	engine := gin.Default()
	// 压缩响应数据，减少网络传输量
	engine.Use(gzip.Gzip(
		gzip.BestCompression, // 压缩级别
		gzip.WithExcludedExtensions([]string{
			".png", ".jpg", ".jpeg", ".gif", ".ico", ".zip", ".gz", ".rar", ".7z", ".mp4", ".mp3",
		}), // 排除的文件扩展名
	))
	// session 数据的存储方式，使用浏览器的 cookie 来保存这些信息，并用密钥进行签名防止篡改。
	store := cookie.NewStore([]byte(config.CookieSecret))

	engine.Use(sessions.Sessions("c", store))
	engine.Use(middleware.CostHandler())
	engine.HTMLRender = loadTemplates("./templates")
	engine.Static("/static", "./static")
	// 创建全局缓存实例
	globalCache := &utils.Cache{
		Data:     make(map[string]interface{}),
		ExpireAt: make(map[string]time.Time),
	}
	// 将缓存实例绑定到 Gin 上下文
	engine.Use(func(c *gin.Context) {
		c.Set("cache", globalCache)
		c.Next()
	})
	// 路由注册
	handler.SetupRouter(injector, engine)
	// 处理 404 错误
	engine.NoRoute(func(c *gin.Context) {
		c.HTML(200, "404.html", gin.H{})
	})
	// 定时任务
	go task.StartPostTask(injector)

	log.Printf("启动http服务,端口:%d,监听请求中...", config.Port)
	engine.Run(fmt.Sprintf(":%d", config.Port))
}

// 时间格式化
func timeAgo(target time.Time) string {
	duration := time.Now().Sub(target)
	if duration < time.Second {
		return "刚刚"
	} else if duration < time.Minute {
		return fmt.Sprintf("%d秒前", duration/time.Second)
	} else if duration < time.Hour {
		return fmt.Sprintf("%d分钟前", duration/time.Minute)
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d小时前", duration/time.Hour)
	} else if duration < 24*time.Hour*365 {
		return fmt.Sprintf("%d天前", duration/(24*time.Hour))
	} else {
		return fmt.Sprintf("%d年前", duration/(24*time.Hour*365))
	}
}

// 自定义模板函数
func templateFun() template.FuncMap {
	return template.FuncMap{
		"timeAgo": timeAgo,
		"html": func(content string) template.HTML {
			return template.HTML(content)
		},
		"css": func(content string) template.CSS {
			return template.CSS(content)
		},
		"js": func(content string) template.JS {
			return template.JS(content)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"dateFormat": func(date time.Time, format string) string {
			return date.Format(format)
		},
		"truncate": func(text string, length int) string {
			if len(text) <= length {
				return text
			}
			return text[:length] + "..."
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
}

// 加载模板
func loadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	layouts, err := filepath.Glob(templatesDir + "/layouts/*.html")
	if err != nil {
		panic(err.Error())
	}
	includes, err := filepath.Glob(templatesDir + "/includes/*.html")
	if err != nil {
		panic(err.Error())
	}

	for _, include := range includes {
		layoutCopy := make([]string, len(layouts))
		copy(layoutCopy, layouts)
		files := append(layoutCopy, include)

		r.AddFromFilesFuncs(filepath.Base(include), templateFun(), files...)
	}
	return r
}

// 初始化系统配置
func initSystem(db *gorm.DB) {
	var systemUserExists int64
	db.Table("tb_user").Where("id=999999999").Count(&systemUserExists)
	if systemUserExists == 0 {
		var systemUser = model.TbUser{
			Username:        "System",
			Password:        "",
			Role:            "",
			Email:           "",
			Bio:             "",
			CommentCount:    0,
			PostCount:       0,
			Status:          "",
			Posts:           nil,
			UpVotedPosts:    nil,
			Points:          0,
			Comments:        nil,
			UpVotedComments: nil,
		}
		systemUser.ID = 999999999
		db.Save(&systemUser)
	}

	var settings model.TbSettings
	if errors.Is(db.First(&settings).Error, gorm.ErrRecordNotFound) {
		saveSettings := vo.SaveSettingsRequest{
			RegMode: "open",
			Css:     "",
			Js:      "",
		}
		settings.Content = model.SaveSettingsRequest(saveSettings)
		db.Save(&settings)
	}

}
