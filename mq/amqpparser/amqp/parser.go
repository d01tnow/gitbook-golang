package amqp

import (
	"bufio"
	"io"
	"time"
)

// Endpoint -
type Endpoint struct {
	Host string
	Port string
}

type ServerEP = Endpoint
type ClientEP = Endpoint

type ConnectionInfo struct {
	ServerEP
	ClientEP
}

// WrappedMessage -
type WrappedMessage struct {
	ConnectionInfo
	ChannelID   uint16
	ConsumerTag string
	Body        []byte
}

// AppPayloadParser -
type AppPayloadParser interface {
	Parse(io.Reader, *ServerEP, *ClientEP, time.Time) error
}

// Parser -
type Parser struct {
	connections map[string]*Connection
	chMsg       chan<- *WrappedMessage
}

// NewParser -
func NewParser(chMsg chan<- *WrappedMessage) *Parser {
	return &Parser{
		connections: make(map[string]*Connection),
		chMsg:       chMsg,
	}
}
func (p *Parser) getConnection(sep *ServerEP, cep *ClientEP) *Connection {
	if conn, ok := p.connections[cep.Port]; ok {
		return conn
	}
	conn := NewConnection(sep, cep, p.chMsg)
	p.connections[cep.Port] = conn
	return conn
}

// Parse -
func (p *Parser) Parse(r io.Reader, serverEP *ServerEP, clientEP *ClientEP, packetTimestamp time.Time) error {
	conn := p.getConnection(serverEP, clientEP)
	buf := bufio.NewReader(r)
	frames := &reader{buf}
	for {
		frame, err := frames.ReadFrame()
		if err != nil {
			// log.Println("read frame failed, err:", err)
			if err == io.EOF {
				return err
			}
		} else {
			conn.demux(frame)
		}
	}
}
