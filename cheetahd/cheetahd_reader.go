package main

import (
	"bufio"
	"errors"
	"io"
)

type CheetahdReader struct {
	content *cheetahdContent
}

// init CheetahdReader
func NewCheetahdReader(content *cheetahdContent) *CheetahdReader {
	return &CheetahdReader{
		content: content,
	}
}

// start cheetahd reader
func (reader *CheetahdReader) StartCheetahdReader(r io.Reader) error {
	buf := bufio.NewReader(r)
	frameReader := &Reader{buf}

	for {
		frame, err := ReadFrame(frameReader)
		if err != nil {
			reader.content.cheetahd.log.Info("cheetahd reader read frame error : %v", err)
			return errors.New("cheetahd reader read frame error")
		}
		reader.content.cheetahd.log.Info("cheetahd reader receieve frame : %v", frame)
	}

	return nil
}
