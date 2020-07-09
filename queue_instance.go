package frame

import (
	"runtime/debug"
)

// 队列结构体
type QueueInstance struct {
	Key       string   //队列key值
	Type      string   //队列类型
	GroupName string
	server    Queue
}

//延迟队列的时候注入方法
var delayHandle func(key string, value string, delayTime int) bool

func SetQueueDelayHandle(f func(key string, value string, delayTime int) bool) {
	delayHandle = f
}

//定义队列类型
const QueueTypeRedis = "redis"

func (queue *QueueInstance) getServer() Queue {
	if queue.server == nil {
		//根据类型获取队列
		if queue.Type == QueueTypeRedis {
			queue.server = &redisQueue{GroupName: queue.GroupName}
		} else {
			msg := map[string]interface{}{
				"error": "获取队列出错",
				"stack": string(debug.Stack()),
			}
			App().Log.Error(msg, LogQueueError)
			panic("获取队列出错")
		}
	}
	return queue.server
}

func (queue *QueueInstance) getKey() string {
	return queue.Key
}

func (queue *QueueInstance) Product(value string, delayTime ...int) bool {
	//延迟队列加入数据库进行暂存,每分钟定时脚本取出塞入队列重新处理
	if len(delayTime) > 0 && delayTime[0] > 0 && delayHandle != nil {
		return delayHandle(queue.Key, value, delayTime[0])
	}
	res := queue.getServer().Product(queue.getKey(), value)
	return res
}

func (queue *QueueInstance) Consume() (string, error) {
	return queue.getServer().Consume(queue.getKey())
}

func (queue *QueueInstance) GetLag() (int, error) {
	return queue.getServer().GetLag(queue.getKey())
}

func (queue *QueueInstance) Empty() bool {
	return queue.getServer().Delete(queue.getKey())
}
