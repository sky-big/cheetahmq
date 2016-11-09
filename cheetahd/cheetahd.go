package main

import (
	"crypto/md5"
	"github.com/Sirupsen/logrus"
	"github.com/sky-big/cheetahmq/util"
	"hash/crc32"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

// cheetahd server struct
type Cheetahd struct {
	sync.RWMutex
	waitGroup          util.WaitGroupWrapper
	name               string `cheetahmq name`
	version            string `cheetahmq server version`
	options            *Options
	log                *logrus.Logger
	tcpListener        net.Listener
	ID                 int64
	exitChan           chan bool
	idChan             chan MessageID
	connectionMap      map[MessageID]*CheetahdConnection
	connectionStopChan chan MessageID
}

// create Cheetahd Struct
func NewCheetahd(options *Options) (cheetahd *Cheetahd) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	h := md5.New()
	io.WriteString(h, hostname)
	defaultID := int64(crc32.ChecksumIEEE(h.Sum(nil)) % 1024)

	cheetahd = &Cheetahd{
		name:               "CheetahMQ Server",
		version:            "0.1.0",
		options:            options,
		ID:                 defaultID,
		connectionMap:      make(map[MessageID]*CheetahdConnection),
		connectionStopChan: make(chan MessageID),
		exitChan:           make(chan bool),
		idChan:             make(chan MessageID),
	}

	// log
	InitLog(cheetahd, options)

	return
}

// Cheetahd Start
func (cheetahd *Cheetahd) Start() {
	cheetahd.log.Infof("CheetahMQ Server Starting")

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

	// start cheetahd main loop
	cheetahd.waitGroup.Wrap(func() {
		cheetahd.mainLoop()
	})

	// start gen id groutine
	cheetahd.waitGroup.Wrap(func() {
		cheetahd.newId()
	})

	// start tcp acceptor groutime by option
	for i := 0; i < cheetahd.options.TcpAcceptorNum; i++ {
		TcpAcceptor := &TcpAcceptor{cheetahdContent}
		index := i
		cheetahd.waitGroup.Wrap(func() {
			TcpAcceptor.AcceptorListen(index, tcpListener)
		})
	}
}

// cheetahd main loop
func (cheetahd *Cheetahd) mainLoop() {
	for {
		select {
		case ConnectionId := <-cheetahd.connectionStopChan:
			cheetahd.unRegisterConnection(ConnectionId)
		case <-cheetahd.exitChan:
			goto exit
		}
	}

exit:
	cheetahd.log.Infof("cheetahd main loop stop")
}

func (cheetahd *Cheetahd) newId() {
	factory := &guidFactory{}
	lastError := time.Unix(0, 0)
	workerID := cheetahd.ID
	for {
		id, err := factory.NewGUID(workerID)
		if err != nil {
			now := time.Now()
			if now.Sub(lastError) > time.Second {
				// only print the error once/second
				cheetahd.log.Infof("ERROR: %s", err)
				lastError = now
			}
			runtime.Gosched()
			continue
		}
		select {
		case cheetahd.idChan <- id.Hex():
		case <-cheetahd.exitChan:
			goto exit
		}
	}

exit:
	cheetahd.log.Infof("Cheetahd ID Generator: closing")
}

// register connection
func (cheetahd *Cheetahd) register(ID MessageID, connection *CheetahdConnection) {
	cheetahd.connectionMap[ID] = connection
}

// unregister connection
func (cheetahd *Cheetahd) unRegisterConnection(ID MessageID) {
	if connection, ok := cheetahd.connectionMap[ID]; ok {
		connection.Close()
		delete(cheetahd.connectionMap, ID)
	}
}

// cheetahmq server exit
func (cheetahd *Cheetahd) Exit() {
	// close listener
	cheetahd.Lock()
	cheetahd.tcpListener.Close()
	cheetahd.Unlock()
	// close all connection
	for ConnectionId, _ := range cheetahd.connectionMap {
		cheetahd.unRegisterConnection(ConnectionId)
	}
	// close cheetahd exit chan
	close(cheetahd.exitChan)
	cheetahd.waitGroup.Wait()
}
