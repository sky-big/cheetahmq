package main

import (
	"bufio"
	"errors"
	"io"
	"net"
)

type CheetahdReader struct {
	content     *cheetahdContent
	frameChan   chan frame
	frameReader *Reader
}

// init CheetahdReader
func NewCheetahdReader(content *cheetahdContent) *CheetahdReader {
	return &CheetahdReader{
		content:   content,
		frameChan: make(chan frame),
	}
}

// start cheetahd reader
func (reader *CheetahdReader) StartCheetahdReader(r net.Conn, exitChan chan bool) error {
	buf := bufio.NewReader(r)
	reader.frameReader = &Reader{buf}

	for {
		// check connection exit
		select {
		// send to chan(connection close need close reader groutine)
		case <-exitChan:
			goto exit
		// nothing to do
		default:
		}
		// recv frame from socket
		frame, err := reader.frameReader.ReadFrame()
		if err != nil {
			if err == io.EOF {
				goto exit
			} else {
				reader.content.cheetahd.log.Infof("cheetahd reader read frame error : %v", err)
				return errors.New("cheetahd reader read frame error")
			}
		}
		reader.content.cheetahd.log.Infof("cheetahd reader receieve frame : %v", frame)
		reader.frameChan <- frame
	}

exit:
	reader.content.cheetahd.log.Infof("cheetahd reader close")
	return nil
}
