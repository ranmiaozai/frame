package frame

import (
	"strconv"
	"sync"
)

//定义redis 计数器
type redisCounter struct {
	redis     *Redis
	redisOnce sync.Once
	GroupName string
}

func (redisCounter *redisCounter) getRedis() *Redis {
	if redisCounter.redis == nil {
		redisCounter.redisOnce.Do(func() {
			redisCounter.redis = &Redis{GroupName: redisCounter.GroupName}
		})
	}
	return redisCounter.redis
}

func (redisCounter *redisCounter) Get(key string) (int, error) {
	res, err := redisCounter.getRedis().Get(key)
	if err != nil {
		return 0, err
	}
	result, _ := strconv.Atoi(res)
	return result, nil
}

func (redisCounter *redisCounter) Set(key string, value int, ttl ...int) bool {
	return redisCounter.getRedis().Set(key, strconv.Itoa(value), ttl...)
}

func (redisCounter *redisCounter) Delete(key string) bool {
	return redisCounter.getRedis().Delete(key)
}

func (redisCounter *redisCounter) Increment(key string, options ...int) (int, error) {
	return redisCounter.getRedis().Increment(key, options...)
}

func (redisCounter *redisCounter) Decrement(key string, options ...int) (int, error) {
	return redisCounter.getRedis().Decrement(key, options...)
}
