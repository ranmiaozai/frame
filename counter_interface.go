package frame
//计数器接口
type Counter interface {
	Get(key string) (int, error)
	Set(key string, value int, ttl ...int) bool
	Delete(key string) bool
	Increment(key string, options ...int) (int, error)
	Decrement(key string, options ...int) (int, error)
}
