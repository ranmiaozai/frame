package frame
//缓存接口
type Cache interface {
	Get(key string) (string, error)
	Set(key string, value interface{}, ttl ...int) bool
	Delete(key string) bool
}