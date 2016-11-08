package main

import (
	"bufio"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/sky-big/cheetahmq/util"
	"io"
	"net"
	"sync"
	"time"
)

type CheetahdConnection struct {
	ID                   MessageID             // connection id
	vhost                string                // cur connection vhost
	sync.RWMutex                               // connection lock
	sendMutex            sync.Mutex            // send msg to client lock
	content              *cheetahdContent      // server info point
	waitGroup            util.WaitGroupWrapper // wait group groutine
	handShakeInfo        string                // client with server handshake info
	reader               *CheetahdReader       // read msg from client groutine info
	writer               *Writer               // write to msg to client
	exitChan             chan bool             // connection exit chan
	status               uint8                 // connection status
	clientProperties     Table                 // client properties
	capabilities         Table                 // capabilities
	auth                 Auther                // about auth
	frameMax             uint32                // amqp frame max size
	channelMax           uint16                // one connection channel max size
	heartbeat            uint16                // server and client heartbeat interval(second)
	lastRecvTime         time.Time             // last recv client msg time
	lastSendTime         time.Time             // last send msg to client time
	timeoutChan          chan bool             // heartbeat timeout chan
	recvHeartBeater      *CheetahdHeartBeat    // recv client msg heartbeat check groutine
	sendHeartBeater      *CheetahdHeartBeat    // send client msg hearbeat check groutine
	heartBeatTimeOutChan chan bool             // recv heartbeat check timeout
	heartBeatSendChan    chan bool             // notice connection send heartbeat msg to client
}

// init one connection
func NewCheetahdConnection(content *cheetahdContent, id MessageID) *CheetahdConnection {
	return &CheetahdConnection{
		content:      content,
		exitChan:     make(chan bool),
		lastRecvTime: time.Now(),
		lastSendTime: time.Now(),
		ID:           id,
	}
}

// start connection
func (connection *CheetahdConnection) StartConnection(conn net.Conn) error {
	connection.content.cheetahd.log.Infof("one connection starting")

	// start connection loop
	connection.waitGroup.Wrap(func() {
		connection.ConnectionMainLoop(conn)
	})

	connection.content.cheetahd.log.Infof("one connection started")

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
		reader.StartCheetahdReader(conn, connection.exitChan)
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
			// update recv client msg time
			if connection.recvHeartBeater != nil {
				connection.recvHeartBeater.lastSendOrRecvTime = time.Now().Unix()
			}
			channelId := frame.channel()
			if channelId == 0 {
				switch detailFrame := frame.(type) {
				case *methodFrame:
					connection.handleMethod0(detailFrame)
				case *heartbeatFrame:
					connection.content.cheetahd.log.Infof("conection recv heartbeat")
				default:
					connection.content.cheetahd.log.Infof("connection recv 0 channelid frame not method frame : %v", frame)
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
					connection.Exit(errors.New("channelid not 0,but recv heartbeat msg"))
				default:
					connection.content.cheetahd.log.Infof("connection recv error frame : %v", frame)
				}
			}
		// heartbeat notice send heartbeat msg send to client
		case <-connection.heartBeatSendChan:
			connection.sendHeartBeartMsgToClient()
			continue
		// heartbeat notice connection timeout
		case <-connection.heartBeatTimeOutChan:
			connection.content.cheetahd.log.Infof("heartbeat timeout, connection exit")
			connection.Exit(errors.New("heartbeat timeout, connection exit"))
		// connection exit
		case <-connection.exitChan:
			goto exit
		}
	}

exit:
	connection.content.cheetahd.log.Infof("cheetahd connection normal exit")
}

// send frame to client
func (connection *CheetahdConnection) sendFrameToClient(frame frame) {
	connection.sendMutex.Lock()
	connection.writer.WriteFrame(frame)
	// update send msg to client time
	if connection.sendHeartBeater != nil {
		connection.sendHeartBeater.lastSendOrRecvTime = time.Now().Unix()
	}
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
	switch connection.status {
	case ConnectionStarting:
		connection.handleStartConnectionOK(methodFrame)
	case ConnectionTuning:
		connection.handleConnectionTuneOK(methodFrame)
	case ConnectionOpening:
		connection.handleConnectionOpen(methodFrame)
	default:
		connection.Exit(errors.New("connection handleMethod0 status error"))
	}
}

// handle start connection ok msg from client
func (connection *CheetahdConnection) handleStartConnectionOK(frame *methodFrame) {
	if startConnectionOKMsg, ok := frame.Method.(*connectionStartOk); ok {
		// init client properties
		connection.clientProperties = startConnectionOKMsg.ClientProperties
		if capabilities, ok := startConnectionOKMsg.ClientProperties["capabilities"]; ok {
			connection.capabilities = capabilities.(Table)
		}
		connection.content.cheetahd.log.Infof("handleStartConnectionOK Respose info : %v", []byte(startConnectionOKMsg.Response))

		auth := NewAuth(startConnectionOKMsg.Mechanism, startConnectionOKMsg.Response)
		if err := auth.GetUserAndPass(); err != nil {
			amqpErrorInfo := NewAmapError("access_refused", startConnectionOKMsg.Response+"auth error", nil)
			errorFrame := MakeExceptionFrame(0, amqpErrorInfo)
			connection.sendFrameToClient(errorFrame)
			connection.Exit(err)
		} else {
			// success get username and passwd
			connection.content.cheetahd.log.WithFields(logrus.Fields{
				"username": auth.GetUserName(),
				"passwd":   auth.GetUserPasswd(),
			}).Info("user name and passwd :")
			if auth.AuthPassCorrectness() {
				// username and passwd correct then send tune msg to client
				connection.sendConnectionTuneMsgToClient()
			} else {
				errorStr := auth.GetUserName() + auth.GetUserPasswd() + "auth user or passwd not match"
				amqpErrorInfo := NewAmapError("access_refused", errorStr, nil)
				errorFrame := MakeExceptionFrame(0, amqpErrorInfo)
				connection.sendFrameToClient(errorFrame)
				connection.Exit(errors.New(errorStr))
			}
		}
	} else {
		connection.Exit(errors.New("handleStartConnectionOK message error"))
	}
}

// send connection tune msg to client
func (connection *CheetahdConnection) sendConnectionTuneMsgToClient() {
	connectionTuneMsg := &connectionTune{
		ChannelMax: uint16(connection.content.cheetahd.options.FrameMax),
		FrameMax:   uint32(connection.content.cheetahd.options.ChannelMax),
		Heartbeat:  uint16(connection.content.cheetahd.options.HeartBeat),
	}
	classId, methodId := connectionTuneMsg.id()
	// send connection tune to client
	connection.sendFrameToClient(&methodFrame{
		ChannelId: 0,
		ClassId:   classId,
		MethodId:  methodId,
		Method:    connectionTuneMsg,
	})
	// setup connection tuning
	connection.status = ConnectionTuning
}

// handle connection tune ok msg
func (connection *CheetahdConnection) handleConnectionTuneOK(method *methodFrame) {
	if tuneOk, ok := method.Method.(*connectionTuneOk); ok {
		if !validateFrameMaxServerAndClientValue(uint32(connection.content.cheetahd.options.FrameMax), tuneOk.FrameMax, frameMinSize) {
			connection.Exit(errors.New("client setup frame max size error"))
		}
		connection.frameMax = tuneOk.FrameMax
		if !validateChannelMaxServerAndClientValue(uint16(connection.content.cheetahd.options.ChannelMax), tuneOk.ChannelMax, ChannelMinSize) {
			connection.Exit(errors.New("client setup channel max size error"))
		}
		connection.channelMax = tuneOk.ChannelMax
		// setup heartbeat
		connection.heartbeat = tuneOk.Heartbeat
		// setup connection status opening
		connection.status = ConnectionOpening
		// start hearbeat groutine
		connection.startHeartBeat()
	} else {
		connection.Exit(errors.New("handleConnectionTuneOK message error"))
	}
}

// handle connection open
func (connection *CheetahdConnection) handleConnectionOpen(method *methodFrame) {
	if connectionOpen, ok := method.Method.(*connectionOpen); ok {
		// setup connection vhost
		connection.vhost = connectionOpen.VirtualHost
		connection.content.cheetahd.log.Infof("connection handleConnectionOpen : %v", connectionOpen)
		// send connection open ok msg to client
		connectionOpenOKMethod := &connectionOpenOk{}
		classId, methodId := connectionOpenOKMethod.id()
		connection.sendFrameToClient(&methodFrame{
			ChannelId: 0,
			ClassId:   classId,
			MethodId:  methodId,
			Method:    connectionOpenOKMethod,
		})
		// setup connection running
		connection.status = ConnectionRunning
	}
}

// start heartbeat
func (connection *CheetahdConnection) startHeartBeat() {
	// start send msg heartbeat check groutine
	connection.heartBeatSendChan = make(chan bool)
	connection.sendHeartBeater = NewHeartBeart(int64(connection.heartbeat), 0, connection.heartBeatSendChan, connection.content)
	connection.waitGroup.Wrap(func() {
		HeartBeatLoop(connection.sendHeartBeater, connection.exitChan)
	})

	// start recv msg heartbeat check groutine
	connection.heartBeatTimeOutChan = make(chan bool)
	connection.recvHeartBeater = NewHeartBeart(int64(connection.heartbeat), 1, connection.heartBeatTimeOutChan, connection.content)
	connection.waitGroup.Wrap(func() {
		HeartBeatLoop(connection.recvHeartBeater, connection.exitChan)
	})
}

// send heartbeat msg to client func
func (connection *CheetahdConnection) sendHeartBeartMsgToClient() {
	heartBeatFrame := &heartbeatFrame{ChannelId: 0}
	connection.sendFrameToClient(heartBeatFrame)
}

// validate framemax server and client value
func validateFrameMaxServerAndClientValue(serverValue uint32, clientValue uint32, minValue int) bool {
	if (clientValue != 0) && (int(clientValue) < minValue) {
		return false
	} else if (serverValue != 0) && (clientValue == 0 || clientValue > serverValue) {
		return false
	}

	return true
}

// validate channelmax server and client value
func validateChannelMaxServerAndClientValue(serverValue uint16, clientValue uint16, minValue int) bool {
	if (clientValue != 0) && (int(clientValue) < minValue) {
		return false
	} else if (serverValue != 0) && (clientValue == 0 || clientValue > serverValue) {
		return false
	}

	return true
}

func handshakeCheck(handshakeStr string) bool {
	if handshakeStr == "AMQP0091" {
		return true
	}

	return false
}

// connection close
func (connection *CheetahdConnection) Close() {
	connection.Exit(errors.New("cheetahd notice connection normal close"))
}

// connection exit
func (connection *CheetahdConnection) Exit(err error) {
	connection.content.cheetahd.log.Infof("connection exit reason : %v", err)
	// unregister from cheetahd
	connection.content.cheetahd.UnRegisterConnection(connection.ID)
	// close heartbeat groutine
	connection.sendHeartBeater.Exit()
	connection.recvHeartBeater.Exit()
	// close self
	close(connection.exitChan)
	// wait group groutine stop
	connection.waitGroup.Wait()
	connection.content.cheetahd.log.Infof("connection exit completely")
}
