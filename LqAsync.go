package LqAsync

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// 任务执行对象
type LqAsync struct {
	Timeout time.Duration          //超时时间
	Count   int                    //任务数
	tasks   map[string]LqAsyncInfo //异步执行所需要的数据
	rwMutex *sync.RWMutex
}

// 任务执行所需要的数据
type LqAsyncInfo struct {
	Handler reflect.Value   //方法地址
	Params  []reflect.Value //参数
}

//创建一个新的任务执行对象
func NewAsync() LqAsync {
	return LqAsync{tasks: make(map[string]LqAsyncInfo), rwMutex: new(sync.RWMutex)}
}

//创建一个新的任务执行对象 老版本的兼容
func NewAsyncOld() LqAsync {
	return NewAsync()
}

//添加任务
//name : 任务名称
//handler：函数方法
//params：函数方法需要的参数
func (a *LqAsync) Addfunc(name string, handler interface{}, params ...interface{}) bool {
	//如果任务已经存在则退出，避免重复添加任务
	if _, e := a.tasks[name]; e {
		return false
	}
	//返回handler的reflect.Value
	handlerValue := reflect.ValueOf(handler)
	//判断是否handler的类型是否是func
	if handlerValue.Kind() == reflect.Func {
		//参数个数
		paramNum := len(params)
		//设置任务的数据
		a.tasks[name] = LqAsyncInfo{
			Handler: handlerValue,
			Params:  make([]reflect.Value, paramNum),
		}
		//设置任务数据的Params
		if paramNum > 0 {
			for k, v := range params {
				a.tasks[name].Params[k] = reflect.ValueOf(v)
			}
		}
		//自增任务数
		a.Count++
		return true
	}

	return false
}

// timeoutRun 任务执行函数
// 在所有任务都运行完成时，将会返回一个map[string][]interface{}的结果。
func (a *LqAsync) TimeoutRun() (map[string][]interface{}, bool) {
	//程序开启多核支持
	runtime.GOMAXPROCS(runtime.NumCPU())
	if a.Count < 1 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.Timeout)
	defer cancel()

	//开启阻塞主线程的执行，直到所有的goroutine执行完成
	wg := sync.WaitGroup{}
	//结果集
	result := make(map[string][]interface{})
	chans := make(chan map[string]interface{}, a.Count)

	wg.Add(1)
	go func(result map[string][]interface{}, chans chan map[string]interface{}) {
		defer wg.Done()
		for {
			if a.Count < 1 {
				break
			}
			select {
			case <-ctx.Done():
				return
			case res := <-chans:
				a.Count--
				a.rwMutex.Lock()
				result[res["name"].(string)] = res["result"].([]interface{})
				a.rwMutex.Unlock()
			}
		}
	}(result, chans)
	//循环任务
	for k, v := range a.tasks {
		wg.Add(1)
		go func(name string, async LqAsyncInfo, chans chan map[string]interface{}, ctx context.Context) {
			defer wg.Done()
			c := make(chan map[string]interface{}, 1)
			go func() {
				result := make([]interface{}, 0)
				//设置返回值
				//name：任务名称
				//result:返回值
				//执行任务函数方法
				values := async.Handler.Call(async.Params)
				//任务返回值
				if valuesNum := len(values); valuesNum > 0 {
					for _, v := range values {
						result = append(result, v.Interface())
					}
				}
				c <- map[string]interface{}{"name": name, "result": result}
			}()
			select {
			case <-ctx.Done():
				return
			case str := <-c:
				chans <- str
				return
			}

		}(k, v, chans, ctx)
	}
	//等待goroutine全部执行完成
	wg.Wait()

	return result, true
}
