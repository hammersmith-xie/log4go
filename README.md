# log4go
在[alecthomas/log4go](github.com/alecthomas/log4go)基础上改的。感谢作者。

## 使用说明
  在example/example.go可以看到例子。
  ```go
  package main

import(
	l4g "github.com/hammersmith-xie/log4go"
	"time"
)

func main(){
	l4g.InitFileLogWriter("gateway","log/")
	l4g.Finest("songshiqi")
	l4g.Error("3Oh no!  %d + %d = %d!", 2, 2, 2+2)

	l4g.Info("all about songshiqi")
	time.Sleep(1*time.Second)
}

  ```
  给InitFileLogWriter传2个string类型参数后即可使用，例子中的"gateway"是项目名，"log/"是可执行文件所在目录的相对路径，为log文件将放置的路径。
  以上参数随实际设置的变化而变化。
