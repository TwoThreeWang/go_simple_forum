package task

import (
	"context"
	"log"
	"time"

	"go_simple_forum/utils"

	"github.com/samber/do"
)

// StartCleanCacheTask 启动定期清理缓存的任务
// ctx 用于控制任务的生命周期
// i 依赖注入容器
func StartCleanCacheTask(ctx context.Context, i *do.Injector) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("缓存清理任务已停止")
			return
		case <-ticker.C:
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Printf("缓存清理任务发生错误: %v", err)
					}
				}()

				cache, err := do.Invoke[*utils.Cache](i)
				if err != nil {
					log.Printf("获取缓存实例失败: %v", err)
					return
				}

				cache.CleanExpired()
				log.Printf("缓存清理完成，当前缓存数量: %d", cache.Size())
			}()
		}
	}
}
