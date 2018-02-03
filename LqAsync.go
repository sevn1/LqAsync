package LqAsync

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// LqAsync 任务执行对象
type LqAsync struct {
	timeout time.Duration          //超时时间
	count   int                    //任务数
	tasks   map[string]LqAsyncInfo //异步执行所需要的数据
}

// 任务执行所需要的数据
type LqAsyncInfo struct {
	Handler reflect.Value   //方法地址
	Params  []reflect.Value //参数
}

//创建一个新的任务执行对象
func NewAsync() LqAsync {
	return LqAsync{tasks: make(map[string]LqAsyncInfo)}
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
		a.count++
		return true
	}

	return false
}

// timeoutRun 任务执行函数
// 在所有任务都运行完成时，将会返回一个map[string][]interface{}的结果。
func (a *LqAsync) timeoutRun() (map[string][]interface{}, bool) {
	//程序开启多核支持
	runtime.GOMAXPROCS(runtime.NumCPU())
	if a.count < 1 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	//开启阻塞主线程的执行，直到所有的goroutine执行完成
	wg := sync.WaitGroup{}
	//添加携程数
	wg.Add(a.count)
	//结果集
	result := make(map[string][]interface{})
	chans := make(chan map[string]interface{}, a.count)

	go func(result map[string][]interface{}, chans chan map[string]interface{}) {
		for {
			if a.count < 1 {
				break
			}
			select {
			case <-ctx.Done():
				break
			case res := <-chans:
				a.count--
				result[res["name"].(string)] = res["result"].([]interface{})
			}
		}
	}(result, chans)
	//循环任务
	for k, v := range a.tasks {
		go func(name string, async LqAsyncInfo, chans chan map[string]interface{}, ctx context.Context) {
			defer wg.Done()
			result := make([]interface{}, 0)
			//设置返回值
			//name：任务名称
			//result:返回值
			defer func(name string, chans chan map[string]interface{}) {
				chans <- map[string]interface{}{"name": name, "result": result}
			}(name, chans)
			//执行任务函数方法
			for {
				select {
				case <-ctx.Done(): //超时则结束
					return
				default:
					values := async.Handler.Call(async.Params)
					//任务返回值
					if valuesNum := len(values); valuesNum > 0 {
						resultItems := make([]interface{}, valuesNum)
						for k, v := range values {
							resultItems[k] = v.Interface()
						}
						result = resultItems
					}
					return
				}
			}

		}(k, v, chans, ctx)
	}
	wg.Wait()
	return result, true
}
