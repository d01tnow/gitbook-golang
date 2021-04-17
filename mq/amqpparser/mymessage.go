package main

import (
	"encoding/binary"
	"time"
	"unsafe"
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
