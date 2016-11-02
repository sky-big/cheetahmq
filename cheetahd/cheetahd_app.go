package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/judwhite/go-svc/svc"
	"github.com/mreiferson/go-options"
)

func cheetahdFlagSet(opts *Options) *flag.FlagSet {
	flagSet := flag.NewFlagSet("cheetahd", flag.ExitOnError)

	flagSet.String("config", opts.Config, "path to config file")
	flagSet.String("tcp-listen-address", opts.TcpListenAddress, "<addr>:<port> to listen on for TCP clients")
	flagSet.String("log-level", "info", "server log level")
	flagSet.Int("tcp-acceptor-num", 2, "tcp listen acceptor groutine num")

	return flagSet
}

type program struct {
	cheetahd *Cheetahd
}

func main() {
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	if env.IsWindowsService() {
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

func (p *program) Start() error {
	opts := NewOptions()
	flagSet := cheetahdFlagSet(opts)
	flagSet.Parse(os.Args[1:])

	var cfg map[string]interface{}
	configFile := flagSet.Lookup("config").Value.String()
	if configFile != "" {
		_, err := toml.DecodeFile(configFile, &cfg)
		if err != nil {
			log.Fatalf("ERROR: failed to load config file %s - %s", configFile, err.Error())
		}
	}
	options.Resolve(opts, flagSet, cfg)

	// init Cheetahd Struct
	p.cheetahd = NewCheetahd(opts)

	// Cheetahd start
	p.cheetahd.Start()

	log.Println("CheetahMQ Server Started")

	return nil
}

func (p *program) Stop() error {
	if p.cheetahd != nil {
		p.cheetahd.Exit()
	}
	return nil
}
