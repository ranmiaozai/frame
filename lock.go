package frame

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

//加锁解锁服务,利用这个可以做一些高并发场景需求
//用法:
//	先加锁,加锁成功运行后续业务(业务结束后解锁,不可再次执行的操作可以不解锁等待自动解锁),加锁失败重试加锁流程(具体重试由业务方自己控制)
type Lock struct {
	GroupName string
	redis     *Redis
	redisOnce sync.Once
	KeyPrefix string
	LockTime  int
}

func (lock *Lock) getRedis() *Redis {
	if lock.redis == nil {
		lock.redisOnce.Do(func() {
			lock.redis = &Redis{GroupName: lock.GroupName}
		})
	}
	return lock.redis
}

//加单据锁
// orderId 单据ID
// lockTime 锁过期时间（秒）
func (lock *Lock) AddLock(orderId string, lockTime ...int) (string, bool) {
	if orderId == "" {
		return "", false
	}
	lockTimeRange := lock.LockTime
	if len(lockTime) > 0 {
		if lockTime[0] > 0 {
			lockTimeRange = lockTime[0]
		}
	}
	//生成唯一锁ID，解锁需持有此ID
	intUniqueLockId := lock.generateUniqueLockId()
	//根据模板，结合单据ID，生成唯一Redis key（一般来说，单据ID在业务中系统中唯一的）
	strKey := lock.KeyPrefix + orderId

	//加锁
	res := lock.getRedis().SetNx(strKey, intUniqueLockId, lockTimeRange)
	if res {
		return intUniqueLockId, true
	}
	return "", false
}

//解单据锁
// orderId 单据ID
// lockId 锁唯一ID
// return bool
func (lock *Lock) ReleaseLock(orderId string, lockId string) bool {
	if orderId == "" || lockId == "" {
		return false
	}
	//根据模板，结合单据ID，生成唯一Redis key（一般来说，单据ID在业务中系统中唯一的）
	strKey := lock.KeyPrefix + orderId

	return lock.getRedis().WatchDelete(strKey, lockId)
}

func (lock *Lock) generateUniqueLockId() string {
	rand.Seed(time.Now().UnixNano())
	str := strconv.Itoa(int(time.Now().Unix())) + strconv.Itoa(os.Getpid()) + strconv.Itoa(rand.Intn(1000))
	return Md5(str)
}
