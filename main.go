package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

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
	var config1 langserver.Config1
	err = yaml.NewDecoder(f).Decode(&config1)
	if err != nil || config1.Version == 2 {
		f, err = os.Open(yamlfile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		err = yaml.NewDecoder(f).Decode(&config)
		if err != nil {
			return nil, fmt.Errorf("can not read configuration: %v", err)
		}
	} else {
		config.Version = config1.Version
		config.Commands = config1.Commands
		config.LogWriter = config1.LogWriter
		languages := make(map[string][]langserver.Language)
		for k, v := range config1.Languages {
			languages[k] = []langserver.Language{v}
		}
		config.Languages = languages
	}
	return &config, nil
}

func main() {
	var yamlfile string
	var logfile string
	var dump bool
	flag.StringVar(&yamlfile, "c", "", "path to config.yaml")
	flag.StringVar(&logfile, "log", "", "logfile")
	flag.BoolVar(&dump, "d", false, "dump configuration")
	flag.Parse()

	if yamlfile == "" {
		dir := os.Getenv("HOME")
		if dir == "" && runtime.GOOS == "windows" {
			dir = filepath.Join(os.Getenv("APPDATA"), "efm-langserver")
		} else {
			dir = filepath.Join(dir, ".config", "efm-langserver")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			log.Fatal(err)
		}
		yamlfile = filepath.Join(dir, "config.yaml")
	}

	config, err := loadConfig(yamlfile)
	if err != nil {
		log.Fatal(err)
	}

	if dump {
		err = yaml.NewEncoder(os.Stdout).Encode(&config)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}
	log.Println("efm-langserver: reading on stdin, writing on stdout")

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		config.LogWriter = f
	}

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
