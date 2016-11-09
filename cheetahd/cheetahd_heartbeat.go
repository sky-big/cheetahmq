package main

import (
	"time"
)

type CheetahdHeartBeat struct {
	name               string
	heartbeatInterval  int64
	retryTimes         uint8
	curRetryTimes      uint8
	checkInterval      time.Duration
	lastSendOrRecvTime int64
	operateChan        chan bool
	content            *cheetahdContent
	exitChan           chan bool
}

// new heartbeat info
func NewHeartBeart(name string, heartbeatInterval int64, retryTimes uint8, operateChan chan bool, content *cheetahdContent) *CheetahdHeartBeat {
	return &CheetahdHeartBeat{
		name:              name,
		heartbeatInterval: heartbeatInterval,
		retryTimes:        retryTimes,
		curRetryTimes:     0,
		checkInterval:     time.Second,
		operateChan:       operateChan,
		content:           content,
		exitChan:          make(chan bool),
	}
}

// heartbeat loop
func HeartBeatLoop(heartbeat *CheetahdHeartBeat) {
	ticker := time.NewTicker(heartbeat.checkInterval)

	for {
		select {
		case <-ticker.C:
			if time.Now().Unix() > (heartbeat.lastSendOrRecvTime + heartbeat.heartbeatInterval) {
				if heartbeat.curRetryTimes >= heartbeat.retryTimes {
					heartbeat.curRetryTimes = 0
					heartbeat.operateChan <- true
				} else {
					heartbeat.curRetryTimes = heartbeat.curRetryTimes + 1
				}
			}
		case <-heartbeat.exitChan:
			goto exit
		}
	}

exit:
	// stop ticker
	ticker.Stop()
	heartbeat.content.cheetahd.log.Infof("heartbeat groutine normal exit")
}

func (hearbeat *CheetahdHeartBeat) Close() {
	close(hearbeat.exitChan)
}
