package provider

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/samber/do"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type AppConfig struct {
	Port         int    `env:"PORT" env-default:"32919"`
	Version      string `env:"VERSION" env-default:"1.0.0"`
	GinMode      string `env:"GIN_MODE" env-default:"debug"`
	DB           string `env:"DB"`
	CookieSecret string `env:"COOKIE_SECRET" env-default:"UbnpjqcvDJ8mDCB"`
	
	// 站点信息
	SiteName string `env:"SiteName" env-default:"竹林"`
	SiteUrl  string `env:"SiteUrl" env-default:"http://localhost:32919"`
	
	// 邮件服务
	EmailApiUrl    string `env:"EmailApiUrl"`
	EmailSender    string `env:"EmailSender"`
	EmailSenderName string `env:"EmailSenderName"`
	EmailPassword  string `env:"EmailPassword"`
	EmailSmtpHost  string `env:"EmailSmtpHost" env-default:"smtp.mail.ru"`
	EmailSmtpPort  int    `env:"EmailSmtpPort" env-default:"587"`
	
	// OAuth配置
	ClientID     string `env:"ClientID"`
	ClientSecret string `env:"ClientSecret"`
	
	// Cloudflare验证
	CFSecretKey string `env:"CFSecretKey"`
	CFVerifyURL string `env:"CFVerifyURL" env-default:"https://challenges.cloudflare.com/turnstile/v0/siteverify"`
}

// NewRepository 数据库连接
func NewRepository(i *do.Injector) (*gorm.DB, error) {
	appConfig := do.MustInvoke[*AppConfig](i)
	fmt.Println(appConfig.DB)
	db, err := gorm.Open(postgres.Open(appConfig.DB), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info),
		TranslateError: true})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// NewAppConfig 加载配置文件
func NewAppConfig(i *do.Injector) (*AppConfig, error) {
	var cfg AppConfig

	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
