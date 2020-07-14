package frame

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type logTpl struct {
	Level string      `json:"level"`
	Msg   interface{} `json:"msg"`
	Time  string      `json:"time"`
	Type  string      `json:"type"`
	Pid   int         `json:"pid"`
}
type logBehaviorTpl struct {
	Msg  interface{} `json:"msg"`
	Time string      `json:"time"`
	Path string      `json:"path"`
}

var errorHandle func(msg string, logType string, logLevel string)

//日志记录结构体
// 通过frame.App().Log 进行使用
type log struct {
	path         string
	behaviorPath string
}

//定义几种错误级别
const LogTypeError = "error"
const LogTypeWarn = "warn"
const LogTypeInfo = "info"
const LogTypeDebug = "debug"
const LogTypeBehavior = "behavior" //行为日志

var myLog *log
var logOnce sync.Once

func getLog() *log {
	logOnce.Do(func() {
		if myLog == nil {
			myLog = &log{}
			log := &struct {
				Log struct {
					Path         string `toml:"path"`
					BehaviorPath string `toml:"behaviorPath"`
				} `toml:"log"`
			}{}
			err := App().Env("app", log)
			if err != nil {
				panic(LogPathError)
			}
			myLog.path = log.Log.Path
			myLog.behaviorPath = log.Log.BehaviorPath
		}
	})
	return myLog
}

func (myLog *log) Error(msg interface{}, contentName string) {
	tpl := &logTpl{
		Level: LogTypeError,
		Msg:   msg,
		Time:  myLog.getLogTime(),
		Type:  contentName,
	}
	myLog.log(tpl)
}
func (myLog *log) Info(msg interface{}, contentName string) {
	tpl := &logTpl{
		Level: LogTypeInfo,
		Msg:   msg,
		Time:  myLog.getLogTime(),
		Type:  contentName,
	}
	myLog.log(tpl)
}
func (myLog *log) Debug(msg interface{}, contentName string) {
	tpl := &logTpl{
		Level: LogTypeDebug,
		Msg:   msg,
		Time:  myLog.getLogTime(),
		Type:  contentName,
	}
	myLog.log(tpl)
}
func (myLog *log) Warn(msg interface{}, contentName string) {
	tpl := &logTpl{
		Level: LogTypeWarn,
		Msg:   msg,
		Time:  myLog.getLogTime(),
		Type:  contentName,
	}
	myLog.log(tpl)
}
func (myLog *log) Behavior(msg interface{}, contentName string) {
	tpl := &logTpl{
		Level: LogTypeBehavior,
		Msg:   msg,
		Time:  myLog.getLogTime(),
		Type:  contentName,
	}
	myLog.log(tpl)
}
func (myLog *log) GetPath() string {
	return myLog.path
}
func (myLog *log) GetBehaviorPath() string {
	return myLog.behaviorPath
}
func (myLog *log) log(tpl *logTpl) {
	var logFile string
	var logMsg string
	if tpl.Level == LogTypeBehavior {
		behaviorTpl := &logBehaviorTpl{
			Msg:  tpl.Msg,
			Time: tpl.Time,
			Path: tpl.Type,
		}
		switch behaviorTpl.Msg.(type) {
		case error:
			tplMsg := fmt.Sprintf("%s", behaviorTpl.Msg)
			behaviorTpl.Msg = tplMsg
		default:

		}
		msg, err := json.Marshal(behaviorTpl)
		logMsg = string(msg) + "\n"
		if err != nil {
			fmt.Println(err)
			return
		}
		logFile = myLog.behaviorPath + "/behavior.log"
	} else {
		tpl.Pid = os.Getpid()
		switch tpl.Msg.(type) {
		case error:
			tplMsg := fmt.Sprintf("%s", tpl.Msg)
			tpl.Msg = tplMsg
		default:

		}
		msg, err := json.Marshal(tpl)
		logMsg = string(msg) + "\n"
		if err != nil {
			fmt.Println(err)
			return
		}
		logFile = myLog.path + "/" + tpl.Level + ".log"
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	_, _ = file.Write([]byte(logMsg))
	//增加一个对外方法可以进行其他操作
	if errorHandle != nil && tpl.Level != LogTypeBehavior {
		errorHandle(logMsg, tpl.Type, tpl.Level)
	}
}

func (myLog *log) getLogTime() string {
	now := time.Now()
	logTime := fmt.Sprintf("%d-%d-%d %d:%d:%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	return logTime
}

//设置错误处理
func SetErrorHandle(f func(msg string, logType string, logLevel string)) {
	errorHandle = f
}
