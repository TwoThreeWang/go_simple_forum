package main

import (
	"embed"
	"encoding/gob"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kingwrcy/hn/handler"
	"github.com/kingwrcy/hn/middleware"
	"github.com/kingwrcy/hn/model"
	"github.com/kingwrcy/hn/provider"
	"github.com/kingwrcy/hn/task"
	"github.com/kingwrcy/hn/utils"
	"github.com/kingwrcy/hn/vo"
	"github.com/samber/do"

	"log"

	"gorm.io/gorm"
)

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

//go:embed static
var staticFS embed.FS

//go:embed templates
var templatesFS embed.FS

func main() {
	// 加载配置文件
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file" + err.Error())
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

	gob.Register(vo.Userinfo{})
	engine := gin.Default()
	engine.Use(gzip.Gzip(gzip.DefaultCompression))
	//store, _ := redis.NewStore(10, "tcp", config.RedisAddress, "", []byte(config.CookieSecret))
	store := cookie.NewStore([]byte(config.CookieSecret))

	engine.Use(sessions.Sessions("c", store))
	engine.Use(middleware.CostHandler())
	engine.HTMLRender = loadLocalTemplates("./templates")
	engine.Static("/static", "./static")
	//if os.Getenv("GIN_MODE") == "release" {
	//	ts, _ := fs.Sub(templatesFS, "templates")
	//	engine.HTMLRender = loadTemplates(ts)
	//	s, _ := fs.Sub(staticFS, "static")
	//	engine.StaticFS("/static", http.FS(s))
	//} else {
	//	engine.HTMLRender = loadLocalTemplates("./templates")
	//	engine.Static("/static", "./static")
	//}
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

	go task.StartPostTask(injector)

	log.Printf("启动http服务,端口:%d,监听请求中...", config.Port)
	engine.Run(fmt.Sprintf(":%d", config.Port))
}

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
		"getStaticPath": func(resource string) string {
			prefix := os.Getenv("STATIC_CDN_PREFIX")
			if prefix == "" {
				return "/static" + resource
			}
			return prefix + resource
		},
	}
}

func loadLocalTemplates(templatesDir string) multitemplate.Renderer {
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

func loadTemplates(templatesDir fs.FS) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	layouts, err := fs.Glob(templatesDir, "layouts/*.html")
	if err != nil {
		panic(err.Error())
	}
	includes, err := fs.Glob(templatesDir, "includes/*.html")
	if err != nil {
		panic(err.Error())
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, include := range includes {
		layoutCopy := make([]string, len(layouts))
		copy(layoutCopy, layouts)
		files := append(layoutCopy, include)
		templateContents := make([]string, len(files))

		for _, f := range files {
			open, err := templatesDir.Open(f)
			if err != nil {
				panic(err)
			}
			buffer, err := io.ReadAll(open)
			if err != nil {
				panic(err)
			}
			templateContents = append(templateContents, string(buffer))
			open.Close()
		}
		r.AddFromStringsFuncs(filepath.Base(include), templateFun(), templateContents...)
	}
	return r
}

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
