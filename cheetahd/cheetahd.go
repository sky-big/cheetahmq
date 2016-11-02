package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/sky-big/cheetahmq/util"
	"net"
	"os"
	"sync"
)

// cheetahd server struct
type Cheetahd struct {
	sync.RWMutex
	waitGroup   util.WaitGroupWrapper
	name        string `cheetahmq name`
	version     string `cheetahmq server version`
	options     *Options
	log         *logrus.Logger
	tcpListener net.Listener
}

// create Cheetahd Struct
func NewCheetahd(options *Options) (cheetahd *Cheetahd) {
	cheetahd = &Cheetahd{
		name:    "CheetahMQ Server",
		version: "0.1.0",
		options: options,
	}

	// log
	InitLog(cheetahd, options)

	return
}

// Cheetahd Start
func (cheetahd *Cheetahd) Start() {
	cheetahd.log.Info("CheetahMQ Server Starting")

	cheetahdContent := &cheetahdContent{cheetahd}
	// start tcp listen
	tcpListener, err := net.Listen("tcp", cheetahd.options.TcpListenAddress)
	if err != nil {
		cheetahd.log.Fatalf("FATAL : listen %s failed reason : %s", cheetahd.options.TcpListenAddress, err)
		os.Exit(1)
	}
	cheetahd.Lock()
	cheetahd.tcpListener = tcpListener
	cheetahd.Unlock()

	// start tcp acceptor groutime by option
	for i := 0; i < cheetahd.options.TcpAcceptorNum; i++ {
		TcpAcceptor := &TcpAcceptor{cheetahdContent}
		index := i
		cheetahd.waitGroup.Wrap(func() {
			TcpAcceptor.AcceptorListen(index, tcpListener)
		})
	}
}

// cheetahmq server exit
func (cheetahd *Cheetahd) Exit() {
	cheetahd.Lock()
	cheetahd.tcpListener.Close()
	cheetahd.Unlock()
	cheetahd.waitGroup.Wait()
}
