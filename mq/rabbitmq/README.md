# Rabbitmq

安装包

```go
go get github.com/streadway/amqp
```



导入包

```go
import (
	"github.com/streadway/amqp"
)
```



## 声明

### 队列



```go
func DeclareQueue(conn *amqp.Connection) error {
  channel, err := conn.Channel
  
}

```



### 交换机

## 发布者
