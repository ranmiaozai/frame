package frame

import (
	"strconv"
	"sync"
)

//定义mc 计数器
type mcCounter struct {
	mc        *memcached
	mcOnce    sync.Once
	GroupName string
}

func (mcCounter *mcCounter) getMc() *memcached {
	if mcCounter.mc == nil {
		mcCounter.mcOnce.Do(func() {
			mcCounter.mc = &memcached{GroupName: mcCounter.GroupName}
		})
	}
	return mcCounter.mc
}

func (mcCounter *mcCounter) Get(key string) (int, error) {
	res, err := mcCounter.getMc().Get(key)
	if err != nil {
		return 0, err
	}
	result, err := strconv.Atoi(res)
	if err != nil {
		if res == "" {
			return 0, nil
		}
		return 0, err
	}
	return result, nil
}

func (mcCounter *mcCounter) Set(key string, value int, ttl ...int) bool {
	return mcCounter.getMc().Set(key, value, ttl...)
}

func (mcCounter *mcCounter) Delete(key string) bool {
	return mcCounter.getMc().Delete(key)
}

func (mcCounter *mcCounter) Increment(key string, options ...int) (int, error) {
	return mcCounter.getMc().Increment(key, options...)
}

func (mcCounter *mcCounter) Decrement(key string, options ...int) (int, error) {
	return mcCounter.getMc().Decrement(key, options...)
}
