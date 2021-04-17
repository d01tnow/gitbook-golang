package rbmq

import (
	"context"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// channelPool -
// 接口归属于使用者
type channelPool interface {
	channel(name, password, vhost string) (*amqp.Channel, error)
}

// Client -
type Client struct {
	Name    string
	Vhost   string
	User    string
	Passwd  string
	channel *amqp.Channel
	cp      channelPool
}

func isBadConn(err error) bool {
	return err == amqp.ErrClosed
}

func isMaxChannelSize(err error) bool {
	return err == amqp.ErrChannelMax
}

func needNewConnection(err error) bool {
	return isBadConn(err) || isMaxChannelSize(err)
}

// Publish -
func (c *Client) Publish(ctx context.Context, exchange, routing string, mandatory, immediate bool, publising *amqp.Publishing) error {

	retry.Do(func() error { return nil })
	// c.channel.Publish(exchange, routing, mandatory, immediate, *publising)
	return errors.Wrap(nil, "not implemented")
}
