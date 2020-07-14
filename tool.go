package frame

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
)

/**
常用函数和方法
*/
func WriteLog(msg interface{}, logType string) {
	App().Log.Error(msg, logType)
}

/**
为字符串字符 \  ' " 加反斜杠
*/
func AddSlashes(str string) string {
	tmpRune := make([]rune, 0)
	strRune := []rune(str)
	for _, ch := range strRune {
		switch ch {
		case []rune{'\\'}[0], []rune{'"'}[0], []rune{'\''}[0]:
			tmpRune = append(tmpRune, []rune{'\\'}[0])
			tmpRune = append(tmpRune, ch)
		default:
			tmpRune = append(tmpRune, ch)
		}
	}
	return string(tmpRune)
}

/**
去掉AddSlashes 加上的反斜杠
*/
func StripSlashes(str string) string {
	dstRune := make([]rune, 0)
	strRune := []rune(str)
	strLength := len(strRune)
	for i := 0; i < strLength; i++ {
		if strRune[i] == []rune{'\\'}[0] {
			i++
		}
		dstRune = append(dstRune, strRune[i])
	}
	return string(dstRune)
}

/**
huo获取主机hostname
*/
func GetHostName() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	return host
}

/**
md5操作
*/
func Md5(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}

//执行系统命令并返回结果,如 res:=ExecShell("ls -al")
func ExecShell(s string) (string, error) {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command("/bin/bash", "-c", s)

	//读取io.Writer类型的cmd.Stdout，再通过bytes.Buffer(缓冲byte类型的缓冲器)将byte类型转化为string类型(out.String():这是bytes类型提供的接口)
	var out bytes.Buffer
	cmd.Stdout = &out

	//Run执行c包含的命令，并阻塞直到完成。  这里stdout被取出，cmd.Wait()无法正确获取stdin,stdout,stderr，则阻塞在那了
	err := cmd.Run()

	return out.String(), err
}

//异步执行系统命令
func ExecShellAsync(s string) error {
	cmd := exec.Command("/bin/bash", "-c", s)
	return cmd.Start()
}


/**
关闭系统所用各种资源
*/
func CloseResource() {
	//默认数据库资源关闭
	closeDB()
	//关闭client中所有闲置连接池
	closeHttpClient()
	//关闭memcached中闲置连接
	closeMemcachedPool()
	//关闭redis中闲置连接
	closeRedis()
}
