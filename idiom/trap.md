# 坑

以下的点, 也不能完全算是 golang 的坑, 需要看好官方文档, 不能想当然.

## iota

`iota` 实际上是常量声明块中当前行的索引运算符，因此，如果首次使用 `iota` 不是常量声明块中的第一行，则初始值将不为零。

```go
const (
	c1 = iota
    c2
)
const (
	c3 = "c3"
    c4
    c5 = iota
    c6
)
const (
	c7 = iota+10
    c8
    c9 = iota
    c10
)
func t() {
    fmt.Println(c1, c2) // 0 1
    fmt.Println(c3, c4, c5, c6) // c3 c3 2 3
    fmt.Println(c7, c8, c9, c10) // 10 11 2 3
}
```



## for range



## slice



## map

**map 的 value 不可寻址** , **通过接口引用的变量也是不可寻址的**

```go
type data struct {
	name string
}
func (d *data)print() {
    fmt.Println("name: ", d.name)
}
func t() {
    // value 是值类型
    m := map[string]data {"x":{"one"}}
    // m["x"].name = "two" // 编译错误: cannot assign to struct field m["x"].name in map
    // m["x"].print() // 编译错误: cannot call pointer method on m["x"]
					  //          cannot take the address of m["x"]
    r := m["x"]
    r.name = "two"
    m["x"] = r
    fmt.Printf("%v\n", m) // map[x:{two}]

    // value 是指针
    m1 := map[string]*data {"x":{"one"}}
    m1["x"].name = "two"
    fmt.Printf("%v\n", m1) // map[x:{two}]
}

```



## interface

### interface {}

```go

func t() {
    var i1 interface{} // 空接口, 类型和值都为 nil
    fmt.Println(i1 == nil) // true

    var data *byte // nil 指针
    i1 = data // i1 类型为 *byte, 值为 nil
    fmt.Println(i1 == nil) // flase
    fmt.Printf("%T, %v\n", i1, i1) // *uint8, <nil>
}

```

### 接口引用的变量是不可寻址的

```go
type data struct {
    name string
}
func (d *data)print() {
    fmt.Println("name: ", d.name)
}

type printer interface {
    print()
}

func t() {
    d1 := data{"d1"}
    d1.print() // d1
    //编译错误: cannot use data literal (type data) as type printer in assignment:
    //        data does not implement printer (print method has pointer receiver)
    // var i1 printer = data{"d2"}
    var i2 printer = &d1
    i2.print() // d1
    m := map[string]data {"x":data{"three"}}
    // 编译错误: cannot call pointer method on m["x"]
    // cannot call pointer method on m["x"]
    // m["x"].print() 
}
```

## defer

defer func 和 go func 启动协程一样, 都是在执行该语句时就完成所有函数参数(包含 接收器)的评估(evaluate).

```go
func t() {
    i := 1
    defer fmt.Println(i*2) // 2
    i = 10
}
type data struct {
    name string
}
func (d *data)print() {
    fmt.Println("name: ", d.name)
}
func t1() {
    datas := []data{{"one"}, {"two"}, {"three"}}
    for _, v := range datas {
        // 一个陷阱是 range 返回的 k, v 变量是复用的. v
        fmt.Printf("&v: %p\n", &v)
        go v.print() 
    }
    time.Sleep(3*time.Second)
}
// 一定会输出 3 条一样的 &v: 0xXXXXXXXX
// 但是, 打印的 name: 不一定一样. 根据 &v 指向的值决定. 
```





## timer

参考:

[使用timer.Reset的正确姿势](https://tonybai.com/2016/12/21/how-to-use-timer-reset-in-golang-correctly/)

[定时器没有正确释放造成资源泄漏](https://www.jianshu.com/p/ca5c81d93d16)

* 调用 Stop() bool 方法可阻止Timer被触发。如果timer已经被触发过或者已经被停止了，那么返回false. 否则返回true。Reset() bool 方法可以重置定时器. 如果timer已经被触发过或者已经被停止了，那么返回false. 否则返回true。但是, 底层 timer.C := make(chan Time, 1) 是有 1 个缓冲区的 channel, Stop() 方法不会关闭这个缓冲区, 如果 Stop() 调用返回 false, 而内部的 channel 中还有未取走的消息.  Reset() 后, 消息还在. 造成误触发. 

  ```go
  //如果要重用 在调用 Stop() 
  go func() {
    timer := time.NewTimer(time.Second * 5)
    LOOP: for {
      // 
      if !timer.Stop() {
        // 这样做是因为: timer 内部 channel 可能没有数据, 造成协程挂起.
        // 即使这样做, 还是会有问题, 见下方描述
        select{
          case <-timer.C: // 取出消息
          default:
        }
      }
      // 这时候timer expire过程中sendTime的执行与“drain channel”是分别在两个goroutine中执行的，谁先谁后，完全依靠runtime调度。上面的看似没有问题的代码，也可能存在问题（当然需要时间粒度足够小，比如ms级的Timer）。
      timer.Reset(time.Second*5)
      select {
        case <-timer.C:
        fmt.Println("timer expired")
        break LOOP
      }
    }  
  }()
  ```

  

* 在高性能场景下，不应该使用time.After，而应该使用New.Timer并在不再使用该Timer后（无论该Timer是否被触发）调用Timer的Stop方法来及时释放资源。 

  * time.After 内部创建了一个 timer 对象. 该对象在 timer 触发后才由 runtime 选择时机 GC.

* linux 下time.Parse 得到的是 UTC 时间, time.Format 的时区是本地时区. 而 windows 下 Parse 和 time.Format 都是本地时区. 所以, 用 time.ParseInLocation(layout, str, location) 代替 Parse , 明确时区.  

* AddDate 如果月对应的天溢出时, 不报错, 而是做标注化. 

  ```go
  package main
  
  import (
    "fmt"
    "time"
  )
  
  func main() {
    t := time.Date(2020, 2, 29, 12, 0, 0, 0, time.Local)
    fmt.Println(t.AddDate(-1, 0, 0)) // 2019-03-01 12:00:00 +0800 CST
    t1 := time.Date(2020, 1, 31, 12, 0, 0, 0, time.Local)
    fmt.Println(t1.AddDate(0, 1, 0)) // 2019-03-02 12:00:00 +0800 CST
  }
  ```

  