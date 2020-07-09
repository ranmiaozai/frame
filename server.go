package frame

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"
)

//web服务器结构体
type server struct {
	port       int //监听端口
	httpServer *http.Server
	router     []func(gin *gin.Engine)
	plugin     []func(gin *gin.Engine)
	pidFile    string
}

//启动
func (server *server) Start() {
	//注册信号监听
	server.registerSignal()

	//记录pid
	server.logPid()

	go func() {
		//服务启动
		server.httpServer = &http.Server{
			Addr:         ":" + strconv.Itoa(server.port),
			Handler:      server.initRoute(),
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 5 * time.Second,
		}

		err := server.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			os.Exit(1)
		}
	}()
	server.shutdown()
}

//重启
func (server *server) Restart() {
	server.Stop(true)
	server.Start()
}

//停止
func (server *server) Stop(restart ...bool) {
	server.kill()
	//阻塞等待接收上个程序结束
	pid := server.getPid()
	fmt.Printf("开始关闭进程,进程id：%d\n", pid)
	if pid != -1 {
		for {
			if err := syscall.Kill(pid, 0); err == nil {
				//继续循环等待
				fmt.Printf("进程关闭中,等待0.5s\n")
				time.Sleep(500 * time.Millisecond)
			} else {
				fmt.Printf("进程关闭状态：%s\n", err.Error())
				break
			}
		}
	}
	if len(restart) <= 0 || !restart[0] {
		os.Exit(0)
	}
}

//信号监听
var stopFlag chan bool

func (server *server) registerSignal() {
	stopFlag = make(chan bool, 0)
	go func() {
		listenSignal := []os.Signal{
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
			syscall.SIGKILL,
			syscall.SIGHUP,
		}
		sig := make(chan os.Signal, 0)
		signal.Notify(sig, listenSignal...)
		<-sig
		server.stop()
	}()
}

func (server *server) stop() {
	stopFlag <- true
}

//关闭
func (server *server) shutdown() {
	<-stopFlag
	defer func() {
		//关闭系统资源
		CloseResource()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.httpServer.Shutdown(ctx)
	if err != nil {
		//关闭报错
		defer func() {
			serverError(err)
		}()
	}
}

func (server *server) kill() {
	pid := server.getPid()
	if pid == -1 {
		return
	}
	sig := syscall.SIGTERM
	proc := new(os.Process)
	proc.Pid = pid
	err := proc.Signal(sig)
	if err != nil {
		defer func() {
			serverError(err)
		}()
	}
	return
}

//记录Pid
func (server *server) logPid() {
	pid := os.Getpid()
	file, err := os.OpenFile(server.pidFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	defer func() {
		serverError(err)
	}()
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(strconv.Itoa(pid))
	if err != nil {
		return
	}
	_ = writer.Flush()
}

//获取Pid
func (server *server) getPid() int {
	file, err := os.Open(server.pidFile)
	defer func() {
		serverError(err)
	}()
	if err != nil {
		return -1
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	pidByte, _, err := reader.ReadLine()
	if err != nil {
		return -1
	}
	pid, err := strconv.Atoi(string(pidByte))
	if err != nil {
		return -1
	}
	return pid
}

func (server *server) initRoute() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()
	for _, f := range server.router {
		f(route)
	}
	for _, f := range server.plugin {
		f(route)
	}
	return route
}

//注册路由
func (server *server) RegisterRoute(f func(gin *gin.Engine)) *server {
	server.router = append(server.router, f)
	return server
}

//注册钩子
func (server *server) RegisterPlugin(f func(gin *gin.Engine)) *server {
	server.plugin = append(server.plugin, f)
	return server
}

func serverError(err error) {
	if err != nil {
		msg := map[string]interface{}{
			"error": err.Error(),
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogServerError)
	}
}
