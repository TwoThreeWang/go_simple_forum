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
	c.mu.Lock()
	defer c.mu.Unlock()
	
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
	c.mu.Lock()
	defer c.mu.Unlock()
	
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

// Clear 清空所有缓存数据
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data = make(map[string]interface{})
	c.ExpireAt = make(map[string]time.Time)
}

// Size 获取缓存中的数据数量
func (c *Cache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.Data)
}

// CleanExpired 清理所有过期的缓存数据
func (c *Cache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, expireTime := range c.ExpireAt {
		if expireTime.Before(now) {
			delete(c.Data, key)
			delete(c.ExpireAt, key)
		}
	}
}

// Exists 检查键是否存在且未过期
func (c *Cache) Exists(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	expireTime, ok := c.ExpireAt[key]
	if !ok {
		return false
	}
	
	if expireTime.Before(time.Now()) {
		delete(c.Data, key)
		delete(c.ExpireAt, key)
		return false
	}
	
	return true
}
