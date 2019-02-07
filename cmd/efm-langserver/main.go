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

func loadConfigs(yamlfile string) (map[string]langserver.Config, error) {
	f, err := os.Open(yamlfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var configs map[string]langserver.Config
	err = yaml.NewDecoder(f).Decode(&configs)
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func main() {
	var yamlfile string
	flag.StringVar(&yamlfile, "c", "config.yaml", "path to config.yaml")
	flag.Parse()

	configs, err := loadConfigs(yamlfile)
	if err != nil {
	}
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}
	log.Println("efm-langserver: reading on stdin, writing on stdout")

	handler := langserver.NewHandler(configs)
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
