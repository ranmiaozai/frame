package frame

import (
	"github.com/BurntSushi/toml"
	"runtime/debug"
	"sync"
)

// 用来读取配置
// 通过frame.App().Env() 方法进行读取

type config struct {
	AppName     string //应用名称
	Environment string //环境变量
	EnvPath     string //配置文件路径(区分环境)
}

const EnvBeta = "beta"
const EnvProduct = "product"
const EnvPre = "pre"
const EnvDevelop = "develop"

func (config *config) setEnv(env string) {
	if env != EnvBeta && env != EnvProduct && env != EnvPre && env != EnvDevelop {
		panic(EnvironmentError)
	}
	config.Environment = env
}

//几个环境判断
func (config *config) IsBeta() bool {
	if config.Environment == EnvBeta {
		return true
	}
	return false
}
func (config *config) IsPre() bool {
	if config.Environment == EnvPre {
		return true
	}
	return false
}
func (config *config) IsProduct() bool {
	if config.Environment == EnvProduct {
		return true
	}
	return false
}
func (config *config) IsDevelop() bool {
	if config.Environment == EnvDevelop {
		return true
	}
	return false
}
func (config *config) IsOther() bool {
	if config.Environment == EnvDevelop ||
		config.Environment == EnvPre ||
		config.Environment == EnvProduct ||
		config.Environment == EnvBeta {
		return false
	}
	return true
}

func (config *config) setAppName(appName string) {
	config.AppName = appName
}

func (config *config) setEnvPath(path string, includeEnv ...bool) {
	if len(includeEnv) > 0 && includeEnv[0] {
		config.EnvPath = path
	} else {
		config.EnvPath = path + "/" + config.Environment
	}
}

//读取配置文件内容
// configFile 配置文件路径，如redis/main
// configStruct 需要读取的结构体
func (config *config) Env(configFile string, configStruct interface{}) error {
	filePath := App().EnvPath + "/" + configFile
	return parseConf(filePath, configStruct)
}

/**
从toml配置文件读取配置
*/
var lock sync.RWMutex

func parseConf(filePath string, configStruct interface{}) error {
	lock.RLock()
	defer lock.RUnlock()

	if filePath == "" {
		return ConfigFileError
	}
	_, err := toml.DecodeFile(filePath+".toml", configStruct)
	if err != nil {
		msg := map[string]interface{}{
			"error": err.Error(),
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogConfigError)
	}
	return err
}
