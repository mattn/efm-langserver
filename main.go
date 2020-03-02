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

const (
	name     = "efm-langserver"
	version  = "0.0.9"
	revision = "HEAD"
)

func main() {
	var yamlfile string
	var logfile string
	var dump bool
	var showVersion bool

	flag.StringVar(&yamlfile, "c", "", "path to config.yaml")
	flag.StringVar(&logfile, "log", "", "logfile")
	flag.BoolVar(&dump, "d", false, "dump configuration")
	flag.BoolVar(&showVersion, "v", false, "Print the version")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}

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

	config, err := langserver.LoadConfig(yamlfile)
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

	if logfile == "" {
		logfile = config.LogFile
	}

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
