package frame

import (
	"runtime/debug"
	"time"
)

//go是协程方式,多个协程资源利用很麻烦,单例会造成很多问题
//go 内存回收非常高效 变量回收很轻松
//所以很多都不考虑单例模式
//除了flag.go获取命令行参数的方法外 其余的都是建立在 App() 调用之后,初始化各种资源和变量定义
func init() {
	//注册mysql执行前操作,支持重载
	SetMysqlBeforeExecute(func(mysql *Mysql) {
		//目前啥也不做,哈哈哈
	})
	//注册mysql执行后操作,支持重载
	SetMysqlAfterExecute(func(mysql *Mysql) {
		//记录慢查询
		nowTime := int(time.Now().Unix())
		runSecond := nowTime - mysql.BeginTime
		if runSecond >= 2 {
			App().Log.Warn(map[string]interface{}{
				"sql":        mysql.GetSql(),
				"run_second": runSecond,
				"config":     mysql.DbGroup.Config,
				"stack":      string(debug.Stack()),
			}, LogMysqlSlow)
		}
	})
	//注册mysql执行中的报错,支持重载
	SetMysqlErrorExecute(func(mysql *Mysql, err error) {
		App().Log.Error(map[string]interface{}{
			"sql":    mysql.GetSql(),
			"config": mysql.DbGroup.Config,
			"error":  err.Error(),
			"stack":  string(debug.Stack()),
		}, LogMysqlError)
	})
}
