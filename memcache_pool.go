package frame

import (
	"github.com/bradfitz/gomemcache/memcache"
	"runtime/debug"
	"sync"
	"time"
)

var mcCacheMap map[string]*memcache.Client
var mcCacheOnce sync.Once
var mcLock sync.Mutex

type memCacheConfig struct {
	Servers        []string `toml:"servers"`
	ConnectTimeout int      `toml:"connect_timeout"`
	MaxIdleConn    int      `toml:"maxIdleConn"`
}

func getMc(mcGroup string) *memcache.Client {
	if mcCacheMap == nil {
		mcCacheOnce.Do(func() {
			mcCacheMap = make(map[string]*memcache.Client)
		})
	}
	if cache, ok := mcCacheMap[mcGroup]; ok {
		return cache
	}
	mcLock.Lock()
	defer mcLock.Unlock()
	mcConfig := &memCacheConfig{}
	err := App().Env(mcGroup, mcConfig)
	if err != nil {
		panic(MemcachedConfigError.Error() + ":" + err.Error())
	}
	serverList := mcConfig.Servers
	mc := memcache.New(serverList...)
	if mc == nil {
		msg := map[string]interface{}{
			"error": "memcached New failed",
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogMemcachedError)
		panic("memcached New failed")
	}
	mc.Timeout = time.Duration(mcConfig.ConnectTimeout) * time.Second
	mc.MaxIdleConns = mcConfig.MaxIdleConn
	mcCacheMap[mcGroup] = mc
	return mc
}

//关闭连接池中所有闲置连接
func closeMemcachedPool() {
	if mcCacheMap == nil {
		return
	}
	for _, v := range mcCacheMap {
		v.MaxIdleConns = 0
	}
}