package transport

import (
	"fmt"
	"io"
	"sync"
)

type ConnectionConfig struct {
	MaxFrameSize int
	ReadKey      Key
	WriteKey     Key
}

func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxFrameSize: DefaultMaxFrameSize,
		ReadKey:      InitialKey,
		WriteKey:     InitialKey,
	}
}

// Connection owns framing state and directional keys. Read and write keys are
// intentionally separate because the negotiated protocol can rotate them at
// different points in the session state machine.
type Connection struct {
	stream io.ReadWriteCloser
	reader *FrameReader
	writer *FrameWriter

	readMu  sync.Mutex
	readKey Key

	writeMu  sync.Mutex
	writeKey Key
}

func NewConnection(stream io.ReadWriteCloser, config ConnectionConfig) (*Connection, error) {
	if stream == nil {
		return nil, fmt.Errorf("nil connection stream")
	}
	reader, err := NewFrameReader(stream, config.MaxFrameSize)
	if err != nil {
		return nil, err
	}
	writer, err := NewFrameWriter(stream, config.MaxFrameSize)
	if err != nil {
		return nil, err
	}
	return &Connection{
		stream:   stream,
		reader:   reader,
		writer:   writer,
		readKey:  config.ReadKey,
		writeKey: config.WriteKey,
	}, nil
}

func (c *Connection) ReadFrame() ([]byte, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	return c.reader.ReadFrame(c.readKey)
}

func (c *Connection) WriteFrame(plain []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.writer.WriteFrame(plain, c.writeKey)
}

func (c *Connection) SetReadKey(key Key) {
	c.readMu.Lock()
	c.readKey = key
	c.readMu.Unlock()
}

func (c *Connection) SetWriteKey(key Key) {
	c.writeMu.Lock()
	c.writeKey = key
	c.writeMu.Unlock()
}

func (c *Connection) Close() error {
	return c.stream.Close()
}
