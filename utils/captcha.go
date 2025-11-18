package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Captcha 验证码结构
type Captcha struct {
	Question string `json:"question"` // 验证码题目，如 "3 + 5 = ?"
	Answer   int    `json:"answer"`   // 正确答案
	ID       string `json:"id"`       // 验证码唯一标识
}

// GenerateCaptchaID 生成唯一的验证码ID
func GenerateCaptchaID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// GenerateMathCaptcha 生成简单的数学加法验证码
func GenerateMathCaptcha() *Captcha {
	// 生成1-10之间的随机数
	num1, _ := rand.Int(rand.Reader, big.NewInt(10))
	num2, _ := rand.Int(rand.Reader, big.NewInt(10))

	a := int(num1.Int64()) + 1 // 确保数字在1-10之间
	b := int(num2.Int64()) + 1

	return &Captcha{
		Question: fmt.Sprintf("%d + %d = ?", a, b),
		Answer:   a + b,
		ID:       GenerateCaptchaID(),
	}
}

// SetCaptchaInSession 将验证码存储在session中
func SetCaptchaInSession(c *gin.Context, captcha *Captcha) {
	session := sessions.Default(c)
	session.Set("captcha_answer", captcha.Answer)
	session.Set("captcha_id", captcha.ID)
	session.Set("captcha_expires", time.Now().Add(5*time.Minute).Unix())
	session.Save()
}

// ValidateCaptcha 验证用户输入的验证码答案
func ValidateCaptcha(c *gin.Context, userAnswer int, captchaID string) bool {
	session := sessions.Default(c)

	// 获取存储的验证码信息
	storedAnswer := session.Get("captcha_answer")
	storedID := session.Get("captcha_id")
	storedExpires := session.Get("captcha_expires")

	// 检查验证码是否存在
	if storedAnswer == nil || storedID == nil || storedExpires == nil {
		return false
	}

	// 检查验证码是否过期
	expires, ok := storedExpires.(int64)
	if !ok || time.Now().Unix() > expires {
		ClearCaptchaFromSession(c) // 清除过期验证码
		return false
	}

	// 验证答案和ID是否匹配
	answerMatch := storedAnswer == userAnswer
	idMatch := storedID == captchaID

	if answerMatch && idMatch {
		// 验证成功后清除验证码，防止重复使用
		ClearCaptchaFromSession(c)
		return true
	}

	return false
}

// ClearCaptchaFromSession 清除session中的验证码
func ClearCaptchaFromSession(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("captcha_answer")
	session.Delete("captcha_id")
	session.Delete("captcha_expires")
	session.Save()
}

// CheckCaptchaExists 检查session中是否存在有效的验证码
func CheckCaptchaExists(c *gin.Context) bool {
	session := sessions.Default(c)

	storedAnswer := session.Get("captcha_answer")
	storedExpires := session.Get("captcha_expires")

	if storedAnswer == nil || storedExpires == nil {
		return false
	}

	// 检查是否过期
	expires, ok := storedExpires.(int64)
	if !ok || time.Now().Unix() > expires {
		ClearCaptchaFromSession(c)
		return false
	}

	return true
}