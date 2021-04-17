package amqp

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Channel -
type Channel struct {
	destructor sync.Once
	m          sync.Mutex // struct field mutex
	confirmM   sync.Mutex // publisher confirms state mutex
	notifyM    sync.RWMutex

	connection *Connection

	rpc       chan message
	consumers *consumers

	id uint16

	// closed is set to 1 when the channel has been closed - see Channel.send()
	closed int32

	// true when we will never notify again
	noNotify bool

	// Channel and Connection exceptions will be broadcast on these listeners.
	closes []chan *Error

	// Listeners for active=true flow control.  When true is sent to a listener,
	// publishing should pause until false is sent to listeners.
	flows []chan bool

	// Listeners for returned publishings for unroutable messages on mandatory
	// publishings or undeliverable messages on immediate publishings.
	returns []chan Return

	// Listeners for when the server notifies the client that
	// a consumer has been cancelled.
	cancels []chan string

	// Allocated when in confirm mode in order to track publish counter and order confirms
	confirms   *confirms
	confirming bool

	// Selects on any errors from shutdown during RPC
	errors chan *Error

	// State machine that manages frame order, must only be mutated by the connection
	recv func(*Channel, frame) error

	// Current state for frame re-assembly, only mutated from recv
	message messageWithContent
	header  *headerFrame
	body    []byte
}

func newChannel(c *Connection, id uint16) *Channel {
	return &Channel{
		connection: c,
		id:         id,
		// rpc:        make(chan message),	// 注释是为了忽略不关心的消息
		consumers: makeConsumers(),
		confirms:  newConfirms(),
		recv:      (*Channel).recvMethod,
		errors:    make(chan *Error, 1),
	}
}
func (ch *Channel) send(msg message) (err error) {

	return
}

// Eventually called via the state machine from the connection's reader
// goroutine, so assumes serialized access.
func (ch *Channel) dispatch(msg message) {
	switch m := msg.(type) {
	case *channelClose:
		// lock before sending connection.close-ok
		// to avoid unexpected interleaving with basic.publish frames if
		// publishing is happening concurrently
		ch.m.Lock()
		ch.send(&channelCloseOk{})
		ch.m.Unlock()
		ch.connection.closeChannel(ch, newError(m.ReplyCode, m.ReplyText))

	case *channelFlow:
		ch.notifyM.RLock()
		for _, c := range ch.flows {
			c <- m.Active
		}
		ch.notifyM.RUnlock()
		ch.send(&channelFlowOk{Active: m.Active})

	case *basicCancel:
		ch.notifyM.RLock()
		for _, c := range ch.cancels {
			c <- m.ConsumerTag
		}
		ch.notifyM.RUnlock()
		ch.consumers.cancel(m.ConsumerTag)

	case *basicReturn:
		ret := newReturn(*m)
		ch.notifyM.RLock()
		for _, c := range ch.returns {
			c <- *ret
		}
		ch.notifyM.RUnlock()

	case *basicAck:
		if ch.confirming {
			if m.Multiple {
				ch.confirms.Multiple(Confirmation{m.DeliveryTag, true})
			} else {
				ch.confirms.One(Confirmation{m.DeliveryTag, true})
			}
		}

	case *basicNack:
		if ch.confirming {
			if m.Multiple {
				ch.confirms.Multiple(Confirmation{m.DeliveryTag, false})
			} else {
				ch.confirms.One(Confirmation{m.DeliveryTag, false})
			}
		}

	case *basicDeliver:
		// 如果没有 consumer , 创建之
		if _, ok := ch.consumers.chans[m.ConsumerTag]; !ok {
			deliveries := make(chan Delivery)
			ch.consumers.add(m.ConsumerTag, deliveries)
			go func() {
				for delivery := range deliveries {
					ch := delivery.Acknowledger.(*Channel)
					if ch.connection != nil && ch.connection.chMsg != nil {
						ch.connection.chMsg <- &WrappedMessage{
							ConnectionInfo: ch.connection.ConnectionInfo,
							ChannelID:      ch.id,
							ConsumerTag:    delivery.ConsumerTag,
							Body:           delivery.Body,
						}
					} else {
						fmt.Printf("缺失连接信息. timestamp: %s, message: %s\n", delivery.Timestamp.Format(time.RFC3339Nano), hex.EncodeToString(delivery.Body))
					}
				}
			}()
		}
		ch.consumers.send(m.ConsumerTag, newDelivery(ch, m))
		// TODO log failed consumer and close channel, this can happen when
		// deliveries are in flight and a no-wait cancel has happened

	default:
		// fmt.Printf("忽略消息: %v\n", msg)
		if ch.rpc != nil {
			ch.rpc <- msg
		}
	}
}

func (ch *Channel) transition(f func(*Channel, frame) error) error {
	ch.recv = f
	return nil
}

func (ch *Channel) recvMethod(f frame) error {
	switch frame := f.(type) {
	case *methodFrame:
		if msg, ok := frame.Method.(messageWithContent); ok {
			ch.body = make([]byte, 0)
			ch.message = msg
			return ch.transition((*Channel).recvHeader)
		}

		ch.dispatch(frame.Method) // termination state
		return ch.transition((*Channel).recvMethod)

	case *headerFrame:
		// drop
		return ch.transition((*Channel).recvMethod)

	case *bodyFrame:
		// drop
		return ch.transition((*Channel).recvMethod)
	}

	panic("unexpected frame type")
}

func (ch *Channel) recvHeader(f frame) error {
	switch frame := f.(type) {
	case *methodFrame:
		// interrupt content and handle method
		return ch.recvMethod(f)

	case *headerFrame:
		// start collecting if we expect body frames
		ch.header = frame

		if frame.Size == 0 {
			ch.message.setContent(ch.header.Properties, ch.body)
			ch.dispatch(ch.message) // termination state
			return ch.transition((*Channel).recvMethod)
		}
		return ch.transition((*Channel).recvContent)

	case *bodyFrame:
		// drop and reset
		return ch.transition((*Channel).recvMethod)
	}

	panic("unexpected frame type")
}

// state after method + header and before the length
// defined by the header has been reached
func (ch *Channel) recvContent(f frame) error {
	switch frame := f.(type) {
	case *methodFrame:
		// interrupt content and handle method
		return ch.recvMethod(f)

	case *headerFrame:
		// drop and reset
		return ch.transition((*Channel).recvMethod)

	case *bodyFrame:
		if cap(ch.body) == 0 {
			ch.body = make([]byte, 0, ch.header.Size)
		}
		ch.body = append(ch.body, frame.Body...)

		if uint64(len(ch.body)) >= ch.header.Size {
			ch.message.setContent(ch.header.Properties, ch.body)
			ch.dispatch(ch.message) // termination state
			return ch.transition((*Channel).recvMethod)
		}

		return ch.transition((*Channel).recvContent)
	}

	panic("unexpected frame type")
}

// func (ch *Channel)Consume(queue string) (<-chan Delivery, error) {

// }

// Ack -
func (ch *Channel) Ack(tag uint64, multiple bool) error {
	ch.m.Lock()
	defer ch.m.Unlock()

	return ch.send(&basicAck{
		DeliveryTag: tag,
		Multiple:    multiple,
	})
}

/*
Nack negatively acknowledges a delivery by its delivery tag.  Prefer this
method to notify the server that you were not able to process this delivery and
it must be redelivered or dropped.

See also Delivery.Nack
*/
func (ch *Channel) Nack(tag uint64, multiple bool, requeue bool) error {
	ch.m.Lock()
	defer ch.m.Unlock()

	return ch.send(&basicNack{
		DeliveryTag: tag,
		Multiple:    multiple,
		Requeue:     requeue,
	})
}

/*
Reject negatively acknowledges a delivery by its delivery tag.  Prefer Nack
over Reject when communicating with a RabbitMQ server because you can Nack
multiple messages, reducing the amount of protocol messages to exchange.

See also Delivery.Reject
*/
func (ch *Channel) Reject(tag uint64, requeue bool) error {
	ch.m.Lock()
	defer ch.m.Unlock()

	return ch.send(&basicReject{
		DeliveryTag: tag,
		Requeue:     requeue,
	})
}
