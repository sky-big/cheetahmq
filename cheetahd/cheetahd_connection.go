package main

import (
	"net"
)

type cheetahdConnection struct {
	content *cheetahdContent
}

// init one connection
func newCheetahdConnection(content *cheetahdContent) *cheetahdConnection {
	return &cheetahdConnection{
		content: content,
	}
}

// start connection
func (connection *cheetahdConnection) startConnection(conn net.Conn) {
	connection.content.cheetahd.log.Info("one connection started")
}
