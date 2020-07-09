package frame

import (
	"sync"
)

//定义mc缓存
type mcCache struct {
	mc        *memcached
	mcOnce    sync.Once
	GroupName string
}

func (mcCache *mcCache) getMc() *memcached {
	if mcCache.mc == nil {
		mcCache.mcOnce.Do(func() {
			mcCache.mc = &memcached{GroupName: mcCache.GroupName}
		})
	}
	return mcCache.mc
}

func (mcCache *mcCache) Get(key string) (string, error) {
	return mcCache.getMc().Get(key)
}

func (mcCache *mcCache) Set(key string, value interface{}, ttl ...int) bool {
	return mcCache.getMc().Set(key, value, ttl...)
}

func (mcCache *mcCache) Delete(key string) bool {
	return mcCache.getMc().Delete(key)
}
