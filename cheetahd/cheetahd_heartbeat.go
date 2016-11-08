package main

import (
	"time"
)

type CheetahdHeartBeat struct {
	heartbeatInterval  int64
	retryTimes         uint8
	curRetryTimes      uint8
	checkInterval      time.Duration
	lastSendOrRecvTime int64
	operateChan        chan bool
	content            *cheetahdContent
}

// new heartbeat info
func NewHeartBeart(heartbeatInterval int64, retryTimes uint8, operateChan chan bool, content *cheetahdContent) *CheetahdHeartBeat {
	return &CheetahdHeartBeat{
		heartbeatInterval: heartbeatInterval,
		retryTimes:        retryTimes,
		curRetryTimes:     0,
		checkInterval:     time.Second,
		operateChan:       operateChan,
		content:           content,
	}
}

// heartbeat loop
func HeartBeatLoop(heartbeat *CheetahdHeartBeat, exitChan chan bool) {
	ticker := time.NewTicker(time.Millisecond * 1000 * heartbeat.checkInterval)

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
		case <-exitChan:
			goto exit
		}
	}

exit:
	heartbeat.content.cheetahd.log.Infof("heartbeat groutine normal exit")
}

func (hearbeat *CheetahdHeartBeat) Exit() {
}
