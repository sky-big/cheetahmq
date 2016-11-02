package main

import (
	"errors"
	"github.com/sky-big/cheetahmq/util"
	"io"
	"net"
	"sync"
)

type CheetahdConnection struct {
	sync.RWMutex
	content       *cheetahdContent
	waitGroup     util.WaitGroupWrapper
	handShakeInfo string
	reader        *CheetahdReader
}

// init one connection
func NewCheetahdConnection(content *cheetahdContent) *CheetahdConnection {
	return &CheetahdConnection{
		content: content,
	}
}

// start connection
func (connection *CheetahdConnection) StartConnection(conn net.Conn) error {
	connection.content.cheetahd.log.Info("one connection starting")

	// connection handshake with client
	var handshakeHead [8]byte
	if _, err := io.ReadFull(conn, handshakeHead[:8]); err != nil {
		return err
	}
	// handshake check
	connection.content.cheetahd.log.Info("handshake info : %v", handshakeHead)
	handshakeStr := string(handshakeHead[:])
	if handshakeCheck(handshakeStr) {
		return errors.New("client handshake fromat mistake")
	}
	// store handshake info
	connection.handShakeInfo = handshakeStr

	// start cheetahd_reader groutine
	reader := NewCheetahdReader(connection.content)
	connection.waitGroup.Wrap(func() {
		reader.StartCheetahdReader(conn)
	})
	connection.reader = reader

	// start connection loop
	connection.waitGroup.Wrap(func() {
		connection.ConnectionMainLoop()
	})

	connection.content.cheetahd.log.Info("one connection started")

	return nil
}

// connection groutine main loop
func (connection *CheetahdConnection) ConnectionMainLoop() {

}

func handshakeCheck(handshakeStr string) bool {
	if handshakeStr == "AMQP0091" {
		return true
	}

	return false
}
