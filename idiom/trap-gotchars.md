# 坑

以下的点, 也不能完全算是 golang 的坑, 需要看好官方文档, 不能想当然.

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

  

## Channel

### 引起 panic 的情况

```go

// 关闭 nil channel
func closeNilChannel() {
  var ch chan struct{}
  close(ch) // panic: close of nil channel
}
// 关闭已经关闭的 channel 
func closeClosedChannel() {
  ch := make(chan struct{})
  close(ch)
  close(ch) // panic: close of closed channel
}
// 写入已经关闭的 channel 
func writeClosedChannel() {
  ch := make(chan struct{}, 1)
  close(ch)
  ch <- struct{}{} // panic: send on closed channel
}

```

### 如何判断 channel 是否关闭 

**没办法直接判断 channel 是否关闭**. 通过读取 channel 的*第二个*返回值为 false , 可以确定 channel 已经关闭.

``` go
func readFromChannel() {
  ch := make(chan int, 10)
  for {
    i, ok := <-ch
    if !ok {
      // 此时, i 是其类型的默认初始化值
      fmt.Println("channel closed!")
      break
    }
    fmt.Println(i)
  }
}
```

## Channel 关闭原则

一个比较适用于大多数情况的关闭原则: 不要从接收端关闭 channel, 也不要在多个并发的发送端关闭 channel. 由只有一个发送端关闭或者多个发送端的最后一个活跃的发送端关闭. 就是说 **始终由发送端关闭** channel.

[参考](https://zhuanlan.zhihu.com/p/297046394)

### 不好的方案

不要尝试在没有同步保护的情况下, 先判断是否关闭再发送. 因为, 判断完成后, chan 关闭了, 发送引发 panic.

* 利用 panic-recover 机制

```go
// 发送
func safeSend(ch chan int, value int) (sent bool) {
  defer func() {
    if recover() != nil {
      sent = false
    }
  }
  // 已经关闭触发 panic
  ch <- value
  return true
}
// 关闭
// 下面方法关闭并不安全, 多个写入端时, 写入已经关闭的的 channel 还是会引发 panic
func CloseChannel(ch chan int) (nilOrAlreadyClosed bool) {
  defer func() {
    if recover() != nil {
      nilOrAlreadyClosed = true
    }
  }()
  // ch 为 nil 或者 ch 已经关闭, 触发 panic
  close(ch)
  return false
}
// 用 sync.Once 关闭 一样会遇到多个写入端, 一个写入端关闭, 其他再次写入已关闭的 chan 触发 panic 的情况
```

* 利用 sync.Mutex 保护. 每次读写都要锁保护, 效率比较低. 不符合 golang 并发程序的写法.

```go
type MyChan struct {
  ch chan int
  closed bool
  mu sync.Mutex
}
func NewMyChan() *MyChan {
  return &MyChan{ch: make(chan int), closed: false}
}
// 下面方法关闭并不安全, 多个写入端时, 写入已经关闭的的 channel 还是会引发 panic
func (mc *MyChan) Close() {
  mc.mu.Lock()
  defer mc.mu.Unlock()
  if !mc.closed {
    close(mc.Ch)
    closed = true
  }
}
func (mc *MyChan) Send(value int) bool {
  mc.mu.Lock()
  defer mc.mu.Unlock()
  if mc,closed {
    return false
  }
  mc.ch <- value
  return true
}
```



### 优雅的关闭方案

### 关闭仅一个发送端使用的 channel

```go
// 这种情况比较简单
// 发送端直接关闭即可, 接收端不关闭
// 接收端保证使用 if v, ok := <-ch; ok 方式读取即可
```

### 关闭多个发送端使用的 channel

```go
// 思路: 做写入端的退出通知, 最后退出的写入端负责关闭, 或者建立一个专门关闭 chan 的 go routine
// 使用一个额外的 chan 做通知发送端, 请停止发送
// 接收端保证使用 if v, ok := <-ch; ok 方式读取即可
// 比如: 利用 context 先调用 ctx, cancel := context.WithCancel(context.TODO()) 的 cancel() 再 close(ch)
// 写入方伪代码.
select {
  case <-ctx.Done():
    // ... exit
    return
  case c <- x:
}

```

多个发送端使用一个 channel 的示例

```go
package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func genValue() func() int {
	rand.Seed(time.Now().UnixNano())

	return func() int {
		return rand.Intn(100)
	}
}

func sender(ctx context.Context, wg *sync.WaitGroup, ch chan<- int, value func() int) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			// exit
			fmt.Println("sender exiting")
			reting
		case ch <- value():
		}
	}
}

func receive(ctx context.Context, wg *sync.WaitGroup, ch <-chan int) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("receiver exiting")
			return
		case i, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println(i)
		}
	}
}
func main() {
	wgSender := &sync.WaitGroup{}
	wgReceiver := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan int, 10)
	senderCount := 3
	receiverCount := 2
	wgSender.Add(senderCount)
	wgReceiver.Add(receiverCount)
	for i := 0; i < senderCount; i++ {
		go sender(ctx, wgSender, ch, genValue())
	}
	for i := 0; i < receiverCount; i++ {
		go receive(ctx, wgReceiver, ch)
	}
	time.AfterFunc(time.Second, func() { cancel() })
	wgReceiver.Wait()
	wgSender.Wait()
	close(ch)
	fmt.Println("Done.")
}

```





