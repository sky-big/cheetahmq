package main

import (
	"bufio"
	"errors"
	"github.com/sky-big/cheetahmq/util"
	"io"
	"net"
	"sync"
)

type CheetahdConnection struct {
	sync.RWMutex                           // connection lock
	sendMutex        sync.Mutex            // send msg to client lock
	content          *cheetahdContent      // server info point
	waitGroup        util.WaitGroupWrapper // wait group groutine
	handShakeInfo    string                // client with server handshake info
	reader           *CheetahdReader       // read msg from client groutine info
	writer           *Writer               // write to msg to client
	exitChan         chan bool             // connection exit chan
	status           uint8                 // connection status
	clientProperties Table                 // client properties
	capabilities     Table                 // capabilities
	userName         string                // username
	passwd           string                // passwd
}

// init one connection
func NewCheetahdConnection(content *cheetahdContent) *CheetahdConnection {
	return &CheetahdConnection{
		content:  content,
		exitChan: make(chan bool),
	}
}

// start connection
func (connection *CheetahdConnection) StartConnection(conn net.Conn) error {
	connection.content.cheetahd.log.Info("one connection starting")

	// start connection loop
	connection.waitGroup.Wrap(func() {
		connection.ConnectionMainLoop(conn)
	})

	connection.content.cheetahd.log.Info("one connection started")

	return nil
}

// connection groutine main loop
func (connection *CheetahdConnection) ConnectionMainLoop(conn net.Conn) {
	// connection handshake with client
	var handshakeHead [8]byte
	if _, err := io.ReadFull(conn, handshakeHead[:8]); err != nil {
		connection.Exit(err)
		return
	}

	// handshake check
	connection.content.cheetahd.log.Info("handshake info : %v", handshakeHead)
	handshakeStr := string(handshakeHead[:])
	if handshakeCheck(handshakeStr) {
		connection.Exit(errors.New("client handshake fromat mistake"))
		return
	}
	// store handshake info
	connection.handShakeInfo = handshakeStr

	// start cheetahd_reader groutine
	reader := NewCheetahdReader(connection.content)
	connection.waitGroup.Wrap(func() {
		reader.StartCheetahdReader(conn)
	})
	connection.reader = reader

	// init writer
	connection.writer = &Writer{bufio.NewWriter(conn)}

	// send start connection msg to client
	connection.sendStartConnectionToClient(handshakeStr)

	// loop handle reader recv message
	for {
		select {
		// receieve frame from reader groutine
		case frame := <-connection.reader.frameChan:
			channelId := frame.channel()
			if channelId == 0 {
				if methodFrame, ok := frame.(*methodFrame); ok {
					connection.handleMethod0(methodFrame)
				} else {
					connection.content.cheetahd.log.Info("connection recv 0 channelid frame not method frame : %v", frame)
				}
			} else {
				switch frame.(type) {
				// handle methodFrame
				case *methodFrame:
				// handle headerFrame
				case *headerFrame:
				// handle bodyFrame
				case *bodyFrame:
				// handle heartbeatFrame
				case *heartbeatFrame:
				default:
					connection.content.cheetahd.log.Info("connection recv error frame : %v", frame)
				}
			}
		// connection exit
		case <-connection.exitChan:
			goto Exit
		}
	}
Exit:
	connection.content.cheetahd.log.Info("cheetahd connection normal exit")
}

// send frame to client
func (connection *CheetahdConnection) sendFrameToClient(frame frame) {
	connection.sendMutex.Lock()
	connection.writer.WriteFrame(frame)
	connection.sendMutex.Unlock()
}

// send start connection msg to client
func (connection *CheetahdConnection) sendStartConnectionToClient(handshakeStr string) {
	startConnectionMsg := &connectionStart{
		VersionMajor:     byte(handshakeStr[5]),
		VersionMinor:     byte(handshakeStr[6]),
		Locales:          string("en_US"),
		Mechanisms:       ALL_AUTH_TYPES,
		ServerProperties: make(map[string]interface{}),
	}
	classId, methodId := startConnectionMsg.id()
	startConnectionMsg.ServerProperties = Table(connection.makeServerProperties())
	startConnectionMsgFrame := &methodFrame{
		ChannelId: 0,
		ClassId:   classId,
		MethodId:  methodId,
		Method:    startConnectionMsg,
	}
	// send to client
	connection.sendFrameToClient(startConnectionMsgFrame)
	// setup connection starting status
	connection.status = ConnectionStarting
}

// make server properties
func (connection *CheetahdConnection) makeServerProperties() (properties map[string]interface{}) {
	properties = make(map[string]interface{})
	properties["capabilities"] = Table(connection.makeServerCapabilities())
	properties["server_version"] = connection.content.cheetahd.version
	properties["platform"] = string("Golang")

	return
}

// make server capabilities
func (connection *CheetahdConnection) makeServerCapabilities() (capabilities map[string]interface{}) {
	capabilities = make(map[string]interface{})
	capabilities["publisher_confirms"] = true
	capabilities["exchange_exchange_bindings"] = true
	capabilities["basic.nack"] = true
	capabilities["consumer_cancel_notify"] = true
	capabilities["connection.blocked"] = true
	capabilities["consumer_priorities"] = true
	capabilities["authentication_failure_close"] = true
	capabilities["per_consumer_qos"] = true

	return
}

// handle channel 0 method
func (connection *CheetahdConnection) handleMethod0(methodFrame *methodFrame) {
	connection.content.cheetahd.log.Info("handleMethod0 : %v", methodFrame)
	switch connection.status {
	case ConnectionStarting:
		connection.handleStartConnectionOK(methodFrame)
	default:
		connection.Exit(errors.New("connection handleMethod0 status error"))
	}
}

// handle start connection ok msg from client
func (connection *CheetahdConnection) handleStartConnectionOK(frame *methodFrame) {
	if startConnectionOKMsg, ok := frame.Method.(*connectionStartOk); ok {
		connection.clientProperties = startConnectionOKMsg.ClientProperties
		if capabilities, ok := startConnectionOKMsg.ClientProperties["capabilities"]; ok {
			connection.capabilities = capabilities.(Table)
		}
		connection.content.cheetahd.log.Info("handleStartConnectionOK info : %v", []byte(startConnectionOKMsg.Response))
		userName, passwd, _ := GetUserAndPass(startConnectionOKMsg.Mechanism, []byte(startConnectionOKMsg.Response))
		connection.content.cheetahd.log.Info("name passwd : %v %v", userName, passwd)
		// TODO auth userName and passwd
		// if err != nil {
		amqpErrorInfo := NewAmapError("access_refused", userName+passwd+"auth error", nil)
		errorFrame := MakeExceptionMethod(0, amqpErrorInfo)
		connection.sendFrameToClient(errorFrame)
		connection.Exit(errors.New("handleStartConnectionOK auth info error"))
		// }
		connection.userName = userName
		connection.passwd = passwd
	} else {
		connection.Exit(errors.New("handleStartConnectionOK message error"))
	}
}

func handshakeCheck(handshakeStr string) bool {
	if handshakeStr == "AMQP0091" {
		return true
	}

	return false
}

// connection exit
func (connection *CheetahdConnection) Exit(err error) error {
	connection.content.cheetahd.log.Info("connection exit reason : %v", err)
	close(connection.exitChan)
	connection.waitGroup.Wait()

	return nil
}
