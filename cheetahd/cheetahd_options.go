package main

import ()

// cheetahd server option info
type Options struct {
	Config           string `config file path`
	TcpListenAddress string `flag:"tcp-listen-address" cfg:"tcp_listen_address"`
	LogLevel         string `flag:"log-level"`
	TcpAcceptorNum   int    `flag:"tcp-acceptor-num"`
	FrameMax         int    `flag:"frame-max" cfg:"frame_max"`
	ChannelMax       int    `flag:"channel-max" cfg:"channel_max"`
	HeartBeat        int    `flag:"heartbeat" cfg:"heartbeat"`
}

func NewOptions() *Options {
	return &Options{
		Config:           "./option/cheetahd.option",
		TcpListenAddress: "0.0.0.0:5672",
	}
}
