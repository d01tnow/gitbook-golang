package amqp

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// Connection -
type Connection struct {
	ConnectionInfo
	rpc      chan message
	m        sync.Mutex
	channels map[uint16]*Channel
	chMsg    chan<- *WrappedMessage
}

// NewConnection -
func NewConnection(sep *ServerEP, cep *ClientEP, chMsg chan<- *WrappedMessage) *Connection {
	return &Connection{
		ConnectionInfo: ConnectionInfo{
			ServerEP: *sep,
			ClientEP: *cep,
		},
		// rpc:      make(chan message), // 注释掉, 忽略连接 method 消息
		channels: make(map[uint16]*Channel),
		chMsg:    chMsg,
	}
}

func (c *Connection) demux(f frame) {
	if f.channel() == 0 {
		c.dispatch0(f)
	} else {
		c.dispatchN(f)
	}
}

func (c *Connection) dispatch0(f frame) {
	switch mf := f.(type) {
	case *methodFrame:
		switch m := mf.Method.(type) {
		case *connectionClose:
			fmt.Println("connection close")
		case *connectionBlocked:
			fmt.Println("connection blocked")
		case *connectionUnblocked:
			fmt.Println("connection unblocked")
		default:
			// fmt.Printf("connection method: %v\n", m)
			if c.rpc != nil {
				c.rpc <- m
			}
		}
	case *heartbeatFrame:
		// kthx - all reads reset our deadline.  so we can drop this
	default:
		// lolwat - channel0 only responds to methods and heartbeats
		// c.closeWith(ErrUnexpectedFrame)
		fmt.Println("error: unexpected frame")
	}
}

func (c *Connection) dispatchN(f frame) {
	c.m.Lock()
	channel := c.channels[f.channel()]
	c.m.Unlock()

	if channel != nil {
		channel.recv(channel, f)
	} else {
		c.dispatchClosed(f)
	}
}

var gid uint16

func genID() uint16 {
	gid++
	return gid
}

func (c *Connection) dispatchClosed(f frame) {
	channel := newChannel(c, f.channel())
	fmt.Println("dispatchClosed, channel id: ", f.channel())
	c.m.Lock()
	c.channels[f.channel()] = channel
	c.m.Unlock()
	channel.recv(channel, f)
}

// Parse -
func (c *Connection) Parse(r io.Reader) error {
	buf := bufio.NewReader(r)
	frames := &reader{buf}
	for {
		frame, err := frames.ReadFrame()
		if err != nil {
			if err == io.EOF {
				return err
			}
		} else {
			c.demux(frame)
		}
	}

}

func (c *Connection) closeChannel(ch *Channel, err error) {

}
