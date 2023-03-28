package frame

import (
	"github.com/gomodule/redigo/redis"
	"strconv"
	"sync"
	"time"
)

type redisGroup struct {
	Master *redis.Pool
	Slaves []*redis.Pool
}

var redisGroupCache map[string]*redisGroup
var redisGroupCacheOnce sync.Once
var redisLock sync.Mutex

type redisHost struct {
	Master *redisHostConfig   `toml:"master"`
	Slaves []*redisHostConfig `toml:"slaves"`
}

type redisHostConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	Password        string `toml:"password"`
	Timeout         int    `toml:"timeout"`
	MaxIdle         int    `toml:"MaxIdle"`
	MaxActive       int    `toml:"MaxActive"`
	IdleTimeout     int    `toml:"IdleTimeout"`
	MaxConnLifetime int    `toml:"MaxConnLifetime"`
}

/**
建立redis连接池
*/
func GetRedis(groupName string) *redisGroup {
	if redisGroupCache == nil {
		redisGroupCacheOnce.Do(func() {
			redisGroupCache = make(map[string]*redisGroup)
		})
	}
	cache, ok := redisGroupCache[groupName]
	if ok {
		return cache
	}
	redisLock.Lock()
	defer redisLock.Unlock()
	redisConfig := &redisHost{}
	err := App().Env(groupName, redisConfig)
	if err != nil {
		panic(RedisConfigError.Error() + ":" + err.Error())
	}
	defer func() {
		redisError(err)
	}()
	masterConfig := redisConfig.Master
	masterPool := &redis.Pool{
		MaxIdle:     masterConfig.MaxIdle,
		MaxActive:   masterConfig.MaxActive,
		IdleTimeout: time.Duration(masterConfig.IdleTimeout) * time.Second,
		Wait:        true, //超过最大连接数就阻塞等待
		MaxConnLifetime: time.Duration(redisConfig.MaxConnLifetime) * time.Second, //连接生命周期
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", masterConfig.Host+":"+
				strconv.Itoa(masterConfig.Port),
				redis.DialPassword(masterConfig.Password),
				redis.DialDatabase(0),
				redis.DialConnectTimeout(time.Duration(masterConfig.Timeout)*time.Second),
				redis.DialReadTimeout(time.Duration(masterConfig.Timeout)*time.Second),
				redis.DialWriteTimeout(time.Duration(masterConfig.Timeout)*time.Second))
			if err != nil {
				if c != nil {
					c.Close()
				}
				return nil, err
			}
			return c, err
		},
	}
	slavesConfig := redisConfig.Slaves
	slavesPool := make([]*redis.Pool, 0)
	for _, slaveConfig := range slavesConfig {
		slave := &redis.Pool{
			MaxIdle:     slaveConfig.MaxIdle,
			MaxActive:   slaveConfig.MaxActive,
			IdleTimeout: time.Duration(slaveConfig.IdleTimeout) * time.Second,
			Wait:        true, //超过最大连接数就阻塞等待
			MaxConnLifetime: time.Duration(slaveConfig.MaxConnLifetime) * time.Second, //连接生命周期
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", slaveConfig.Host+":"+
					strconv.Itoa(slaveConfig.Port),
					redis.DialPassword(slaveConfig.Password),
					redis.DialDatabase(0),
					redis.DialConnectTimeout(time.Duration(slaveConfig.Timeout)*time.Second),
					redis.DialReadTimeout(time.Duration(slaveConfig.Timeout)*time.Second),
					redis.DialWriteTimeout(time.Duration(slaveConfig.Timeout)*time.Second))
				if err != nil {
					if c != nil {
						c.Close()
					}
					return nil, err
				}
				return c, err
			},
		}
		slavesPool = append(slavesPool, slave)
	}
	redisGroupCache[groupName] = &redisGroup{Master: masterPool, Slaves: slavesPool}
	return redisGroupCache[groupName]
}

func closeRedis() {
	if redisGroupCache == nil {
		return
	}
	for _, cache := range redisGroupCache {
		_ = cache.Master.Close()
		for _, v := range cache.Slaves {
			_ = v.Close()
		}
	}
}
