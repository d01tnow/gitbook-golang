package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/streadway/amqp"
)

// MyMessage -
type MyMessage struct {
	id    uint32
	start time.Time
}

func newMyMessage(id uint32) *MyMessage {
	return &MyMessage{
		id:    id,
		start: time.Now(),
	}
}

// Marshal -
func (m MyMessage) Marshal() ([]byte, error) {
	startbuf, err := m.start.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, int(unsafe.Sizeof(m.id))+len(startbuf))
	binary.BigEndian.PutUint32(buf, m.id)
	copy(buf[unsafe.Sizeof(m.id):], startbuf)
	return buf, nil
}

// Unmarshal -
func (m *MyMessage) Unmarshal(b []byte) error {
	m.id = binary.BigEndian.Uint32(b)
	return m.start.UnmarshalBinary(b[unsafe.Sizeof(m.id):])
}

// ID -
func (m MyMessage) ID() uint32 {
	return m.id
}

// Start -
func (m MyMessage) Start() time.Time {
	return m.start
}

func amqpURL(username, password, host, port, vhost string) string {
	u := fmt.Sprintf("amqp://%s:%s@%s:%s/%s", username, password, host, port, url.PathEscape(vhost))
	fmt.Println(u)
	return u
}

func fatalError(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err.Error())
	}
}

func publish(ctx context.Context) {
	defer wg.Done()
	fmt.Println("Publishing")
	url := amqpURL("usr1", "usr1-pwd", "192.168.50.38", "5672", "/")
	conn, err := amqp.Dial(url)
	fatalError("failed to connect", err)
	defer conn.Close()

	ch, err := conn.Channel()
	fatalError("failed to get channel", err)
	defer ch.Close()
	queueName := "qu1"
	// q, err := ch.QueueDeclare(
	// 	queueName, // queue name
	// 	true,  // durable
	// 	false, // delete when unused
	// 	false, // exclusive
	// 	false, // no-wait
	// 	nil,   // arguments
	// )
	// fatalError("failed to declare queue", err)
	var cnt uint32
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			body, _ := newMyMessage(cnt).Marshal()
			err = ch.Publish(
				"",        // exchange, 默认交换机是 direct 类型
				queueName, // routing key
				false,     // mandatory. true 时表示在没有绑定匹配的队列时发布失败
				false,     // immediate. true 时表示没有消费者时发布失败.
				amqp.Publishing{
					DeliveryMode: amqp.Persistent, // delivery mode 消息也需要持久化
					ContentType:  "text/plain",
					Body:         body,
				},
			)
			// 处理异常关闭
			// handle Exception (504) Reason: "channel/connection is not open"
			fatalError("failed to publish", err)
			// runtime.Gosched()
			time.Sleep(time.Second)
			cnt++
			if cnt%1000 == 0 {
				fmt.Println("published: ", cnt, "elapsed: ", time.Since(start))
				start = time.Now()
			}
		}
	}

}
func consumer(ctx context.Context, conn *amqp.Connection, id int) {
	ch, err := conn.Channel()
	fatalError("failed to get channel", err)
	defer ch.Close()
	ch.Qos(1, 0, false)
	fmt.Printf("[%02d] Consuming\n", id)
	msgs, err := ch.Consume(
		"qu1",                           // queue name
		fmt.Sprintf("consumer%02d", id), // consumer name
		false,                           // auto ack
		false,                           // exclusive
		false,                           // no-local
		false,                           // no-wait
		nil,                             // arguments
	)
	fatalError("failed to consume message", err)

	cnt := 0
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				fmt.Println("message channel closed")
				return
			}
			var mymsg MyMessage
			err := mymsg.Unmarshal(msg.Body)
			fatalError("failed to unmarshal message", err)
			cnt++
			if cnt%1000 == 0 {
				fmt.Printf("[%02d] consumed: %d, elapsed: %v\n", id, cnt, time.Since(mymsg.Start()))
			}
			// 单个确认
			err = msg.Ack(false)
			fatalError("failed to consume message", err)
		}
	}
}

var id int

func consume(ctx context.Context) {
	url := amqpURL("usr1", "usr1-pwd", "192.168.50.38", "5672", "/")
	conn, err := amqp.Dial(url)
	fatalError("failed to connect", err)
	defer conn.Close()

	id++
	go consumer(ctx, conn, id)
	<-ctx.Done()
	wg.Done()
}

var publishFlag bool

func init() {
	flag.BoolVar(&publishFlag, "p", false, "Publish the message")

}

var wg sync.WaitGroup

func main() {

	flag.Parse()
	fmt.Println("args:", flag.Args(), "Narg: ", flag.NArg())

	runtime.GOMAXPROCS(runtime.NumCPU())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(2)
	go publish(ctx)
	go consume(ctx)
	// go consume(ctx)
	fmt.Println("To exit press CTRL+C")
	sg := <-sig
	fmt.Println(sg)
	fmt.Println("CPU:", runtime.NumCPU())
	cancel()
	wg.Wait()
}
