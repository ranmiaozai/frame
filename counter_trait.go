package frame

import "runtime/debug"

const CounterTypeMc = "mc"
const CounterTypeRedis = "redis"


//计数器基础结构体 一般放于另外结构体里面当匿名属性以便实现继承
//用法介绍
//  type UserCounter struct{
//     frame.CounterTrait
//  }
//  func GetUserCounter() *UserCounter {
//		counter:=&UserCounter{}
//      counter.Type=frame.CounterTypeRedis
//      counter.Group="redis/main"
//      counter.Ttl=60
//      counter.PreFixKey="userInfo"
//      return counter
//  }
type CounterTrait struct {
	counter   Counter
	Type      string
	Group     string
	Ttl       int
	PreFixKey string
}

func (counterTrait *CounterTrait) GetCounter() Counter {
	if counterTrait.counter != nil {
		return counterTrait.counter
	}
	if counterTrait.Type == CounterTypeMc {
		counterTrait.counter = &mcCounter{
			GroupName: counterTrait.Group,
		}
	} else if counterTrait.Type == CounterTypeRedis {
		counterTrait.counter = &redisCounter{
			GroupName: counterTrait.Group,
		}
	} else {
		msg := map[string]interface{}{
			"error": CounterError,
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogCounterError)
		panic(CounterError)
	}
	return counterTrait.counter
}

func (counterTrait *CounterTrait) Key(key string) string {
	return counterTrait.PreFixKey + key
}

func (counterTrait *CounterTrait) Get(key string) (int, error) {
	return counterTrait.GetCounter().Get(counterTrait.Key(key))
}

func (counterTrait *CounterTrait) Increment(key string, options ...int) (int, error) {
	return counterTrait.GetCounter().Increment(counterTrait.Key(key), options...)
}

func (counterTrait *CounterTrait) Decrement(key string, options ...int) (int, error) {
	return counterTrait.GetCounter().Decrement(counterTrait.Key(key), options...)
}

func (counterTrait *CounterTrait) Set(key string, value int, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{counterTrait.Ttl}
	}
	return counterTrait.GetCounter().Set(counterTrait.Key(key), value, ttl...)
}

func (counterTrait *CounterTrait) Delete(key string) bool {
	return counterTrait.GetCounter().Delete(counterTrait.Key(key))
}
