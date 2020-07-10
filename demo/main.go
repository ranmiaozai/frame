package main

import (
	"frame"
)

func main()  {
	frame.App().Init("develop","Test","/www/go/my_frame/demo/env")
	redis:=&frame.Redis{GroupName: "redis/main"}
	redis.Delete("lll")
}