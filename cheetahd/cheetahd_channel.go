package main

type CheetahdChannel struct {
	vhost *CheetahdVhost // cur channel vhost info
	state uint8          // channel state
	ID    uint16         // channel ID
}
