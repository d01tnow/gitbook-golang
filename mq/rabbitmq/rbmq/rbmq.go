package rbmq

import (
	"fmt"
	"math/rand"
	"net/url"
	"sync"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	defaultPort = 5601
)

var (
	defaultVHost = url.PathEscape("/")
)

type noCopy struct{}

func (n *noCopy) Lock()   {}
func (n *noCopy) Unlock() {}

// QueueProperty -
type QueueProperty struct {
	Name             string
	Durable          bool
	deleteWhenUnused bool
	Exclusive        bool
	NoWait           bool
	args             map[string]interface{}
}

// HostPort -
type HostPort struct {
	Host string
	Port uint16
}

// Rbmq -
type Rbmq interface {
	NewClient(user, passwd, vhost string) (*Client, error)
}

type rbmq struct {
	noCopy    noCopy
	hostPorts []HostPort
	m         sync.Mutex
	conns     []*amqp.Connection
	chClosed  chan struct{} // 关闭通道
}

// NewRbmq -
func NewRbmq(hostPorts []HostPort) Rbmq {
	r := &rbmq{}
	copy(r.hostPorts, hostPorts)
	return r
}

var _ Rbmq = (*rbmq)(nil)
var _ channelPool = (*rbmq)(nil)

// NewClient -
func (r *rbmq) NewClient(user, passwd, vhost string) (*Client, error) {
	return nil, errors.New("not implemented")
}

func (r *rbmq) channel(user, passwd, vhost string) (*amqp.Channel, error) {
	if len(r.hostPorts) == 0 {
		return nil, errors.New("missing host ports")
	}
	// 测试过程使用固定值
	rand.Seed(30)
	// 正式时使用
	// rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(r.hostPorts))
	url := amqpURL(user, passwd, r.hostPorts[i].Host, fmt.Sprintf("%d", r.hostPorts[i].Port), vhost)
	conn, err := amqp.DialConfig(url, amqp.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to dial")
	}
	return conn.Channel()
}

func (r *rbmq) Close() error {
	var err error
	for i := range r.conns {
		if r.conns[i] != nil {
			err = r.conns[i].Close()
			if err != nil {
				err = errors.Wrap(err, "failed to close connection")
			}
			r.conns[i] = nil
		}
	}
	return err
}

func amqpURL(username, password, host, port, vhost string) string {
	u := fmt.Sprintf("amqp://%s:%s@%s:%s/%s", username, password, host, port, url.PathEscape(vhost))
	fmt.Println(u)
	return u
}
