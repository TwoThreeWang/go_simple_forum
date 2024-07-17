/**
 * @Author: wangcheng
 * @Author: job_wangcheng@163.com
 * @Date: 2024/7/12 下午1:47
 * @Description: 通知工具
 */

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// Notify 定义通知工具接口
type Notify interface {
	Send(toUser, subject, content string) string
}

// Email 实现通知方法
type Email struct{}

// MailPost 邮件发送结构体
type MailPost struct {
	From struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"from"`
	Personalizations []struct {
		To []struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"to"`
	} `json:"personalizations"`
	Subject string `json:"subject"`
	Content []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"content"`
}

// Send Email 实现 Send 方法
func (e Email) Send(toUser, subject, content string) string {
	// 创建请求体数据
	EmailApiUrl := os.Getenv("EmailApiUrl")
	EmailSender := os.Getenv("EmailSender")
	EmailSenderName := os.Getenv("EmailSenderName")
	content += "<br><br>Thanks,<br>The ZhuLin Team."
	postData := fmt.Sprintf(`{
  "from": {
      "email": "%s",
      "name": "%s"
  },
  "personalizations": [
    {
      "to": [
        {
          "email": "%s",
          "name": "%s"
        }
      ]
    }
  ],
  "subject": "%s",
  "content": [
    {
      "type": "text/html",
      "value": "%s"
    }
  ]
}`, EmailSender, EmailSenderName, toUser, toUser, subject, content)

	// 将格式化后的字符串解析为MailPost结构体
	var mailPost MailPost
	err := json.Unmarshal([]byte(postData), &mailPost)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return "系统错误，邮件发送失败！"
	}

	// 将结构体编码为 JSON
	jsonData, err := json.Marshal(mailPost)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return "系统错误，邮件发送失败！"
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", EmailApiUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Email send error:", err)
		return "系统错误，邮件发送失败！"
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Email send error:", err)
		return "系统错误，邮件发送失败！"
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Email result error:", err)
		return "系统错误，邮件发送失败！"
	}

	// 打印响应状态码和内容
	fmt.Println("Email Send Response Status:", resp.Status)
	fmt.Println("Email Send Response Body:", string(body))
	return "Success"
}
