package frame

import (
	"sync"
)

//定义redis队列
type redisQueue struct {
	redis     *Redis
	redisOnce sync.Once
	GroupName string
}

func (redisQueue *redisQueue) getRedis() *Redis {
	if redisQueue.redis == nil {
		redisQueue.redisOnce.Do(func() {
			redisQueue.redis = &Redis{GroupName: redisQueue.GroupName}
		})
	}
	return redisQueue.redis
}

func (redisQueue *redisQueue) Product(key string, value string) bool {
	return redisQueue.getRedis().RPush(key, value)
}

func (redisQueue *redisQueue) Consume(key string) (string, error) {
	return redisQueue.getRedis().LPop(key)
}

func (redisQueue *redisQueue) GetLag(key string) (int, error) {
	return redisQueue.getRedis().LLen(key)
}

func (redisQueue *redisQueue) Delete(key string) bool {
	return redisQueue.getRedis().Delete(key)
}
