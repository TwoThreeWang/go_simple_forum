package utils

import (
	"sync"
	"time"
)

// Cache 内存缓存
type Cache struct {
	mu       sync.Mutex
	Data     map[string]interface{}
	ExpireAt map[string]time.Time
}

// Get 获取缓存数据
func (c *Cache) Get(key string) (interface{}, bool) {
	value, ok := c.Data[key]
	if ok {
		// 检查是否过期
		if c.ExpireAt[key].Before(time.Now()) {
			delete(c.Data, key)
			delete(c.ExpireAt, key)
			return nil, false
		}
	}
	return value, ok
}

// Set 设置缓存数据
func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	c.Data[key] = value
	c.ExpireAt[key] = time.Now().Add(duration)
}

// Delete 删除缓存数据
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Data, key)
	delete(c.ExpireAt, key)
}
