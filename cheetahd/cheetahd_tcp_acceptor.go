package main

import (
	"fmt"
	"net"
	"runtime"
	"strings"
)

type TcpAcceptor struct {
	content *cheetahdContent
}

// tcp acceptor listen
func (acceptor *TcpAcceptor) AcceptorListen(index int, listener net.Listener) {
	acceptor.content.cheetahd.log.Info(fmt.Sprintf("TCPAcceptor%d: listening on %s", index, listener.Addr()))

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				acceptor.content.cheetahd.log.Info(fmt.Sprintf("NOTICE: temporary Accept() failure - %s", err))
				runtime.Gosched()
				continue
			}
			// theres no direct way to detect this error because it is not exposed
			if !strings.Contains(err.Error(), "use of closed network connection") {
				acceptor.content.cheetahd.log.Info(fmt.Sprintf("ERROR: listener.Accept() - %s", err))
			}
			break
		}
		// next step start connection
		connection := NewCheetahdConnection(acceptor.content)
		if err := connection.StartConnection(clientConn); err != nil {
			acceptor.content.cheetahd.log.Info("ERROR: NewCheetahdConnection error : %s", err)
			break
		}
	}

	acceptor.content.cheetahd.log.Info(fmt.Sprintf("TCP: closing %s", listener.Addr()))
}
