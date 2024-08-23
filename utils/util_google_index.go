/**
 * @Author: wangcheng
 * @Author: job_wangcheng@163.com
 * @Date: 2024/8/23 上午9:49
 * @Description: 提交网址到 google
 */

package utils

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/indexing/v3"
	"google.golang.org/api/option"
	"os"
)

func Submit2Google(hrefs []string) error {
	ctx := context.Background()
	dir, err := os.Getwd()
	fmt.Println(dir)
	secretFile := ".google_index.json"
	srv, err := indexing.NewService(ctx, option.WithCredentialsFile(secretFile))
	if err != nil {
		return errors.New("error create google index api service")
	}
	for _, href := range hrefs {
		notification := indexing.UrlNotification{
			Type: "URL_UPDATED",
			Url:  href,
		}
		_, err = srv.UrlNotifications.Publish(&notification).Do()
		if err != nil {
			fmt.Printf("error submit url %s to google.%s\n", href, err)
		} else {
			fmt.Printf("success submit %s to google", href)
		}
	}
	return nil
}
