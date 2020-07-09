package frame

// 队列接口
type Queue interface {
	Product(key string, value string) bool
	Consume(key string) (string,error)
	GetLag(key string) (int,error)
	Delete(key string) bool
}