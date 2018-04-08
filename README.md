# grpool

根据github.com/ivpusic/grpool修改的 Goroutine池
修改job类型，以支持func带参数

## Sample

**sample.go**
```golang
package grpool

import (
	"fmt"
	"runtime"
	"time"

	"github.com/sevn1/grpool"
)

func first() {
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	// number of workers, and size of job queue
	pool := grpool.NewPool(100, 50)

	// release resources used by pool
	defer pool.Release()

	// submit one or more jobs to pool
	for i := 0; i < 10; i++ {
		count := i

		pool.AddFunc(functest,i)
	}

	// dummy wait until jobs are finished
	time.Sleep(1 * time.Second)
}

func functest(c interface{}){
	fmt.Println(c)
}