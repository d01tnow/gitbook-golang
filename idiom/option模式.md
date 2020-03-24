# Option 模式

[Functional Options Pattern][1] . 属性很多或属性易变时采用的一种模式.

优点: 优雅地设置多个属性. 适合默认设置, 属性多, 属性易变等情况. 扩展方便. 

缺点: 增加了复杂度, 属性少或变化较小时不要采用该方式.

方法:

1. 定义配置选项 option, option 是一个函数, 参数是被设置的对象指针(例子中的 Client).
2. 定义多个设置参数的函数, 函数的参数是被设置的对象的某一属性类型, 返回值是 option.
3. 定义创建对象的函数, 函数的参数是不定长的 option .

``` go
type Option func(c *Client)
func WithTimeout(timeout int) Option {
  return func(c *Client) {
    c.timeout = timeout
  }
}
func WithHealthCheck(healthCheck func() bool) Option {
  return func(c *Client) {
    c.healthCheckFunc = healthCheck
  }
}

func NewClient(opts ...Option) *Client {
  // 使用默认值创建 Client
  client := &Client{
    timeout: 10,
    healthCheckFunc: func() bool { return true },
    
  }
  for _, opt := range opts {
    opt(client)
  }
  return client
}
```





[1]: <https://halls-of-valhalla.org/beta/articles/functional-options-pattern-in-go,54/>