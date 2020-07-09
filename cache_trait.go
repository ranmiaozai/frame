package frame

import (
	"runtime/debug"
)

const CacheTypeMc = "mc"
const CacheTypeRedis = "redis"

//缓存基础结构体 一般放于另外结构体里面当匿名属性以便实现继承
//用法介绍
//  type UserCache struct{
//     frame.CacheTrait
//  }
//  func GetUserCache() *UserCache {
//		cache:=&GetUserCache{}
//      cache.Type=frame.CacheTypeRedis
//      cache.Group="redis/main"
//      cache.Ttl=60
//      cache.PreFixKey="userInfo"
//      return cache
//  }
type CacheTrait struct {
	cache     Cache
	Type      string //缓存类型 mc或者redis
	Group     string //需要的配置资源,如 redis/main
	Ttl       int    //默认缓存时长(秒)
	PreFixKey string //缓存key前缀
}

func (cacheTrait *CacheTrait) getCache() Cache {
	if cacheTrait.cache != nil {
		return cacheTrait.cache
	}
	if cacheTrait.Type == CacheTypeMc {
		cacheTrait.cache = &mcCache{
			GroupName: cacheTrait.Group,
		}
	} else if cacheTrait.Type == CacheTypeRedis {
		cacheTrait.cache = &redisCache{
			GroupName: cacheTrait.Group,
		}
	} else {
		msg := map[string]interface{}{
			"error": CacheError.Error(),
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogCacheError)
		panic(CacheError)
	}
	return cacheTrait.cache
}

//返回key的真实key
func (cacheTrait *CacheTrait) Key(key string) string {
	return cacheTrait.PreFixKey + key
}

func (cacheTrait *CacheTrait) Get(key string) (string, error) {
	return cacheTrait.getCache().Get(cacheTrait.Key(key))
}

func (cacheTrait *CacheTrait) Set(key string, value string, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{cacheTrait.Ttl}
	}
	return cacheTrait.getCache().Set(cacheTrait.Key(key), value, ttl...)
}

func (cacheTrait *CacheTrait) Delete(key string) bool {
	return cacheTrait.getCache().Delete(cacheTrait.Key(key))
}
