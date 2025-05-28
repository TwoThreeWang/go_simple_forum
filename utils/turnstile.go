package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// TurnstileResponse represents the response from Cloudflare's siteverify API
type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

// VerifyTurnstileToken validates a Turnstile token with Cloudflare's siteverify API
// secretKey: your Turnstile secret key
// token: the response token from the Turnstile widget
// remoteIP: optional, the visitor's IP address
func VerifyTurnstileToken(c *gin.Context, token, remoteIP string) (*TurnstileResponse, error) {
	// 获取网站配置
	secretKey := os.Getenv("CFSecretKey")
	turnstileSiteVerifyURL := os.Getenv("CFVerifyURL")
	if secretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	// Prepare form data
	form := url.Values{}
	form.Add("secret", secretKey)
	form.Add("response", token)
	if remoteIP != "" {
		form.Add("remoteip", remoteIP)
	}

	// Make POST request to siteverify API
	resp, err := http.Post(
		turnstileSiteVerifyURL,
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send verification request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse response
	var result TurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// 3. 根据验证结果处理业务逻辑
	if result.Success {
		return &result, nil
	} else {
		// 验证失败，可能是机器人或可疑行为
		return nil, fmt.Errorf("Turnstile 验证失败。错误码: %v\n", result.ErrorCodes)
	}
}