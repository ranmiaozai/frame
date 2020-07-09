package frame

import (
	"os"
	"strings"
	"sync"
)

// 获取命令行参数
// 只支持  go run test.go -environment=pre -appName=api 这种方式
func GetFlag(key string, defaultVal ...interface{}) interface{} {
	if flagParams == nil {
		flagParse()
	}
	res, ok := flagParams[key]
	if !ok {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return nil
	}
	return res
}

var flagParams map[string]interface{}
var flagOnce sync.Once

func flagParse() {
	flagOnce.Do(func() {
		flagParams = make(map[string]interface{})
		for k, v := range os.Args {
			if k > 0 {
				v = strings.TrimLeft(v, "-")
				strArr := strings.Split(v, "=")
				if len(strArr) > 1 {
					flagParams[strArr[0]] = strArr[1]
				} else {
					flagParams[strArr[0]] = ""
				}
			}
		}
	})
}