package main

import (
	"github.com/Sirupsen/logrus"
)

// init log
func InitLog(cheetahd *Cheetahd, options *Options) {
	cheetahd.log = logrus.New()
	// init log level
	logLevel, _ := logrus.ParseLevel(options.LogLevel)
	cheetahd.log.Level = logLevel
}
