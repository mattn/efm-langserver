package main

import (
	"context"
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/mattn/efm-langserver/langserver"
	"github.com/sourcegraph/jsonrpc2"
)

func loadConfig(yamlfile string) (*langserver.Config, error) {
	f, err := os.Open(yamlfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config langserver.Config
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {
	var yamlfile string
	flag.StringVar(&yamlfile, "c", "config.yaml", "path to config.yaml")
	flag.Parse()

	config, err := loadConfig(yamlfile)
	if err != nil {
	}
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}
	log.Println("efm-langserver: reading on stdin, writing on stdout")

	handler := langserver.NewHandler(config)
	var connOpt []jsonrpc2.ConnOpt
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		handler, connOpt...).DisconnectNotify()
	log.Println("efm-langserver: connections closed")
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
