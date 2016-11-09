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
	exitChan    chan bool
}

// init CheetahdReader
func NewCheetahdReader(content *cheetahdContent) *CheetahdReader {
	return &CheetahdReader{
		content:   content,
		frameChan: make(chan frame),
		exitChan:  make(chan bool),
	}
}

// start cheetahd reader
func (reader *CheetahdReader) StartCheetahdReader(conn net.Conn) error {
	buf := bufio.NewReader(conn)
	reader.frameReader = &Reader{buf}

	for {
		// check reader exit
		select {
		case <-reader.exitChan:
			goto exit
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
	conn.Close()
	reader.exitChan = nil
	reader.content.cheetahd.log.Infof("cheetahd reader close")
	return nil
}

// stop
func (reader *CheetahdReader) Close() {
	if reader.exitChan != nil {
		reader.exitChan <- true
	}
}
