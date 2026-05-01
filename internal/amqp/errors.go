package amqp

import "errors"

var (
	ErrNotConnected = errors.New("amqp not connected")
	ErrClosed       = errors.New("amqp client closed")
)
