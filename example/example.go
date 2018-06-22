package main

import(
	l4g "github.com/hammersmith-xie/log4go"
	"time"
)

func main(){
	l4g.InitFileLogWriter("gateway","log/",true)
	l4g.Finest("songshiqi")
	l4g.Error("3Oh no!  %d + %d = %d!", 2, 2, 2+2)

	l4g.Info("all about songshiqi")
	time.Sleep(1*time.Second)
}