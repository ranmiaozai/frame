package frame

import "errors"

//自定义错误
//方便将来多语言修改

var ConfigFileError = errors.New("配置文件路径错误")
var CacheError = errors.New("缓存配置错误")
var EnvironmentError = errors.New("环境变量错误,只能是beta|product|pre|develop")
var CounterError = errors.New("计数配置错误")
var DbError = errors.New("数据库配置错误")
var HttpError = errors.New("http配置错误")
var HttpFailError = errors.New("http错误")
var LogPathError = errors.New("log路径设置错误")
var MemcachedConfigError = errors.New("memcached配置错误")
var RedisConfigError = errors.New("redis配置错误")
var DbAllowError = errors.New("目前还未支持")
var DbHandleError = errors.New("数据库操作错误")
