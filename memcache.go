package frame

import (
	"github.com/bradfitz/gomemcache/memcache"
	"runtime/debug"
	"sync"
)

// memcached 基础结构体  实现了很多mc的基础方法
// 需要不断新增和完善
// 不直接对外服务 通过定义新的结构体继承来对外提供服务
type memcached struct {
	mc        *memcache.Client
	mcOnce    sync.Once
	GroupName string
}

func (mc *memcached) getPool() *memcache.Client {
	if mc.mc == nil {
		mc.mcOnce.Do(func() {
			mc.mc = getMc(mc.GroupName)
		})
	}
	return mc.mc
}

func (mc *memcached) Get(key string) (string, error) {
	result, err := mc.getPool().Get(key)
	defer func() {
		mcError(err)
	}()
	if err != nil {
		if err == memcache.ErrCacheMiss {
			err = nil
			return "", nil
		}
		return "", err
	}
	return string(result.Value), nil
}

func (mc *memcached) Set(key string, value interface{}, ttl ...int) bool {
	ttlTime := 0
	if len(ttl) > 0 {
		ttlTime = ttl[0]
	}
	err := mc.getPool().Set(&memcache.Item{Key: key, Value: []byte(convertToString(value)), Expiration: int32(ttlTime)})
	defer func() {
		mcError(err)
	}()
	if err != nil {
		return false
	}
	return true
}

func (mc *memcached) Delete(key string) bool {
	err := mc.getPool().Delete(key)
	defer func() {
		mcError(err)
	}()
	if err != nil {
		if err == memcache.ErrCacheMiss {
			err = nil
			return true
		}
		return false
	}
	return true
}

func (mc *memcached) Increment(key string, options ...int) (int, error) {
	step := 1
	if len(options) > 0 {
		step = options[0]
	}
	res, err := mc.getPool().Increment(key, uint64(step))
	defer func() {
		mcError(err)
	}()
	if err != nil {
		if err != memcache.ErrCacheMiss {
			return 0, err
		}
		mc.Set(key, 0)
		res, err := mc.getPool().Increment(key, uint64(step))
		if err != nil {
			return 0, err
		}
		return int(res), nil
	}
	return int(res), nil
}

func (mc *memcached) Decrement(key string, options ...int) (int, error) {
	step := 1
	if len(options) > 0 {
		step = options[0]
	}
	res, err := mc.getPool().Decrement(key, uint64(step))
	defer func() {
		mcError(err)
	}()
	if err != nil {
		if err != memcache.ErrCacheMiss {
			return 0, err
		}
		return 0, nil
	}
	return int(res), nil
}

func mcError(err error) {
	if err != nil && err != memcache.ErrCacheMiss {
		msg := map[string]interface{}{
			"error": err.Error(),
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogMemcachedError)
	}
}
