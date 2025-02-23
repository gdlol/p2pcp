package transfer

import "io"

type ChannelStream struct {
	stream io.ReadWriteCloser
	Done   chan struct{}
}

func (c *ChannelStream) Read(p []byte) (n int, err error) {
	return c.stream.Read(p)
}

func (c *ChannelStream) Write(p []byte) (n int, err error) {
	return c.stream.Write(p)
}

func (c *ChannelStream) Close() error {
	c.Done <- struct{}{}
	return c.stream.Close()
}

func NewChannelStream(stream io.ReadWriteCloser) *ChannelStream {
	return &ChannelStream{
		stream: stream,
		Done:   make(chan struct{}),
	}
}
