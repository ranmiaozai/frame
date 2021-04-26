package frame

import (
	"frame/Test"
	"github.com/gin-gonic/gin"
	"runtime"
	"sync"
)

type app struct {
	Log    *log
	server *server
	config
}

var appObj *app
var appOnce sync.Once

// 该封装包的主要方法,在使用该包的大部分方法都需要先初始化这个方法,一般在入口的地方进行调用
func App() *app {
	if appObj != nil {
		return appObj
	}
	appOnce.Do(func() {
		appObj = &app{}
	})
	return appObj
}

// App方法之后需要初始化的一些信息
// environment 环境变量
// appName 应用名称
// envPath 配置文件目录(不包含环境变量)
func (app *app) Init(environment string, appName string, envPath string) *app {
	//设置多核心cpu并行
	cpuNum := runtime.NumCPU() //获得当前设备的cpu核心数
	runtime.GOMAXPROCS(cpuNum) //设置需要用到的cpu数量

	app.setEnv(environment)
	app.setAppName(appName)
	app.setEnvPath(envPath)
	//初始化日志
	app.Log = getLog()
	Test.Abcd()
	return app
}

var serverOnce sync.Once
// 服务器程序,需要web服务器的时候调用返回一个server实例
// port 监听的端口
// 服务守护进程id文件
func (app *app) Server(port int, pidFile ...string) *server {
	if app.server != nil {
		return app.server
	}
	serverOnce.Do(func() {
		server := &server{
			port:   port,
			router: make([]func(gin *gin.Engine), 0),
		}
		//设置脚本pid进程文件
		if len(pidFile) > 0 {
			server.pidFile = pidFile[0]
		} else {
			server.pidFile = app.Log.path + "/server_" + app.AppName + "_pid.lock"
		}
		app.server = server
	})
	return app.server
}
