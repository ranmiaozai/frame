package frame

import (
	"github.com/gomodule/redigo/redis"
	"math/rand"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// redis结构体
type Redis struct {
	redisPool     *redisGroup
	redisPoolOnce sync.Once
	GroupName     string
}

var redisReadMethod = []string{
	"get", "exists", "mget", "hget", "hlen", "hkeys", "hvals", "hgetall", "hexists",
	"hmget", "lindex", "lget", "llen", "lsize", "lrange", "lgetrange", "scard", "ssize", "sdiff", "sinter",
	"sismember", "scontains", "smembers", "sgetmembers", "srandmember", "sunion", "zcard", "zsize",
	"zcount", "zrange", "zrangebyscore", "zrevrangebyscore", "zrangebylex", "zrank", "zrevrank", "zrevrange",
	"zscore", "zunion",
}

func (redisObj *Redis) getMaster() *redis.Pool {
	redisObj.initPool()
	return redisObj.redisPool.Master
}

func (redisObj *Redis) getSlave() *redis.Pool {
	redisObj.initPool()
	rand.Seed(time.Now().UnixNano())
	return redisObj.redisPool.Slaves[rand.Intn(len(redisObj.redisPool.Slaves))]
}

func (redisObj *Redis) initPool() {
	if redisObj.redisPool == nil {
		redisObj.redisPoolOnce.Do(func() {
			redisObj.redisPool = GetRedis(redisObj.GroupName)
		})
	}
}

func (redisObj *Redis) getPool(method string) *redis.Pool {
	for _, v := range redisReadMethod {
		if v == strings.ToLower(method) {
			return redisObj.getSlave()
		}
	}
	return redisObj.getMaster()
}

func (redisObj *Redis) Get(key string) (string, error) {
	c := redisObj.getPool("get").Get()
	defer c.Close()
	r, err := redis.String(c.Do("Get", key))
	defer func() {
		redisError(err)
	}()
	if err != nil {
		if err == redis.ErrNil {
			err = nil
			return "", nil
		}
		return "", err
	}
	return r, nil
}

func (redisObj *Redis) Set(key string, value string, ttl ...int) bool {
	ttlTime := -1
	if len(ttl) > 0 && ttl[0] > 0 {
		ttlTime = ttl[0]
	}
	c := redisObj.getPool("set").Get()
	defer c.Close()
	var err error
	defer func() {
		redisError(err)
	}()
	if ttlTime > -1 {
		_, err = c.Do("SET", key, value, "EX", ttlTime)
		if err != nil {
			return false
		}
	} else {
		_, err = c.Do("SET", key, value)
		if err != nil {
			return false
		}
	}
	return true
}

func (redisObj *Redis) SetNx(key string, value string, ttl ...int) bool {
	ttlTime := -1
	if len(ttl) > 0 && ttl[0] > 0 {
		ttlTime = ttl[0]
	}
	//加锁（通过Redis SetNx指令实现，从Redis 2.6.12开始，通过set指令可选参数也可以实现SetNx，同时可原子化地设置超时时间）
	c := redisObj.getPool("set").Get()
	defer c.Close()
	var err error
	defer func() {
		redisError(err)
	}()
	var res interface{}
	if ttlTime > -1 {
		res, err = c.Do("SET", key, value, "EX", ttlTime, "NX")
		if err != nil {
			return false
		}
	} else {
		res, err = c.Do("SET", key, value, "NX")
		if err != nil {
			return false
		}
	}
	if res != nil && res.(string) == "OK" {
		return true
	}
	return false
}

func (redisObj *Redis) WatchDelete(key string, value string) bool {
	//监听Redis key防止在【比对lock id】与【解锁事务执行过程中】被修改或删除，提交事务后会自动取消监控，其他情况需手动解除监控
	c := redisObj.getPool("delete").Get()
	defer c.Close()
	res, err := c.Do("Watch", key)
	if err != nil || res.(string) != "OK" {
		return false
	}
	res, err = redis.String(c.Do("Get", key))
	if err != nil {
		return false
	}
	if res == value {
		_, _ = c.Do("MULTI")
		_, _ = c.Do("Del", key)
		_, _ = c.Do("EXEC")
		return true
	}
	_, _ = c.Do("unwatch")
	return true
}

func (redisObj *Redis) Delete(key string) bool {
	c := redisObj.getPool("delete").Get()
	defer c.Close()
	_, err := c.Do("Del", key)
	defer func() {
		redisError(err)
	}()
	if err != nil {
		return false
	}
	return true
}

func (redisObj *Redis) Increment(key string, options ...int) (int, error) {
	step := 1
	if len(options) > 0 {
		step = options[0]
	}
	c := redisObj.getPool("increment").Get()
	defer c.Close()
	res, err := c.Do("INCRBY", key, step)
	defer func() {
		redisError(err)
	}()
	if err != nil {
		return 0, err
	}
	return int(res.(int64)), nil
}

func (redisObj *Redis) Decrement(key string, options ...int) (int, error) {
	step := 1
	if len(options) > 0 {
		step = options[0]
	}
	c := redisObj.getPool("decrement").Get()
	defer c.Close()
	res, err := c.Do("DECRBY", key, step)
	defer func() {
		redisError(err)
	}()
	if err != nil {
		return 0, err
	}
	return int(res.(int64)), nil
}

func (redisObj *Redis) LLen(key string) (int, error) {
	c := redisObj.getPool("llen").Get()
	defer c.Close()
	r, err := c.Do("llen", key)
	defer func() {
		redisError(err)
	}()
	if err != nil {
		return 0, err
	}
	return int(r.(int64)), nil
}

func (redisObj *Redis) RPush(key string, value string) bool {
	c := redisObj.getPool("rpush").Get()
	defer c.Close()
	_, err := c.Do("Rpush", key, value)
	defer func() {
		redisError(err)
	}()
	if err != nil {
		return false
	}
	return true
}

func (redisObj *Redis) LPop(key string) (string, error) {
	c := redisObj.getPool("lpop").Get()
	defer c.Close()
	r, err := redis.String(c.Do("Lpop", key))
	defer func() {
		redisError(err)
	}()
	if err != nil {
		if err == redis.ErrNil {
			err = nil
		}
		return "", err
	}
	return r, nil
}

func redisError(err error) {
	if err != nil {
		msg := map[string]interface{}{
			"error": err.Error(),
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogRedisError)
	}
}
