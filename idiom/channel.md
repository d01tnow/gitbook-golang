# Channel

* 写入或读取 nil channel 会阻塞当前 go routine

``` go
// 关闭的 channel 不为 nil
func isClosedChanNil() {
  ch := make(chan struct{})
  close(ch) // 不能多次关闭, 否则 panic. 也不能关闭 nil channel
  fmt.Printf("closed chan is nil: %+v\n", ch == nil)
}

// 读取带缓存区的 channel, 如果缓存区有数据, 关闭之后还可以正常读取
func readFromClosedBufferedChannel() {
  ch := make(chan int, 10)
  for i := 0; i< 10; i++ {
    ch <- i
  }
  close(ch)
  for {
    // 必须读取完才能退出
    // 如何在读取完前退出, 见下面的函数
    if i, ok := <-ch; ok {
      fmt.Println(i)
    } else {
      break
    }
  }
}

// 读取带缓存区的 channel, 如果缓存区有数据, 关闭之后还可以正常读取
// 如何在关闭后就退出读取
func breakReadFromClosedBufferedChannel() {
  // 使用 context
  ctx, cancel := context.WithCancel(context.Background())
  ch := make(chan int, 10)
  for i := 0; i< 10; i++ {
    ch <- i
  }
  cancel() // 先退出后关闭更好些
  close(ch)
  // cancel() // 先关闭后退出
exit:
  for {
    // 必须读取完才能退出
    // 如何在读取完前退出, 见下面的函数
    select {
      case <-ctx.Done():
      	break exit
      case i, ok := <-ch:
        fmt.Println(i, " ok: ", ok)
    }
  }
  fmt.Println("exit.")
}


```

## 如何