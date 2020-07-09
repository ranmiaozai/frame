package frame

import (
	"sync"
)

//定义redis缓存
type redisCache struct {
	redis     *Redis
	redisOnce sync.Once
	GroupName string
}

func (redisCache *redisCache) getRedis() *Redis {
	if redisCache.redis == nil {
		redisCache.redisOnce.Do(func() {
			redisCache.redis = &Redis{GroupName: redisCache.GroupName}
		})
	}
	return redisCache.redis
}

func (redisCache *redisCache) Get(key string) (string, error) {
	return redisCache.getRedis().Get(key)
}

func (redisCache *redisCache) Set(key string, value interface{}, ttl ...int) bool {
	return redisCache.getRedis().Set(key, value.(string), ttl...)
}

func (redisCache *redisCache) Delete(key string) bool {
	return redisCache.getRedis().Delete(key)
}
