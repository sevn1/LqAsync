# LqAsync

Wait和Cancel两种并发控制方式结合开发的一个并发处理程序

## Sample

**sample.go**
```golang
package main

import (
	"fmt"
	"time"
	"github.com/sevn1/LqAsync"
)

func main() {
	a := NewAsync()
	a.timeout = 2 * time.Second
	a.Addfunc("functest1", functest1)
	a.Addfunc("functest2", functest2)
	s, ok := a.timeoutRun()
	fmt.Println(s)
	fmt.Println(ok)
}

func functest1() string {
	time.Sleep(1 * time.Second)
	return "result1"
}
func functest2() string {
	time.Sleep(3 * time.Second)
	return "result2"
}

```
**log Print**
```
map[functest1:[result1]]
true
```