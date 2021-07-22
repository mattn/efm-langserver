package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"github.com/mattn/efm-langserver/langserver"
	"github.com/sourcegraph/jsonrpc2"
)

const (
	name    = "efm-langserver"
	version = "0.0.36"
)

var revision = "HEAD"

func main() {
	var yamlfile string
	var logfile string
	var loglevel int
	var dump bool
	var showVersion bool
	var quiet bool

	flag.StringVar(&yamlfile, "c", "", "path to config.yaml")
	flag.StringVar(&logfile, "logfile", "", "logfile")
	flag.IntVar(&loglevel, "loglevel", 1, "loglevel")
	flag.BoolVar(&dump, "d", false, "dump configuration")
	flag.BoolVar(&showVersion, "v", false, "Print the version")
	flag.BoolVar(&quiet, "q", false, "Run quieter")
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
	} else {
		_, err := os.Stat(yamlfile)
		if err != nil {
			log.Fatal(err)
		}
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

	if quiet {
		log.SetOutput(ioutil.Discard)
	}

	log.Println("efm-langserver: reading on stdin, writing on stdout")

	if logfile == "" {
		logfile = config.LogFile
	}
	if config.LogLevel > 0 {
		loglevel = config.LogLevel
	}

	var connOpt []jsonrpc2.ConnOpt

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		config.Logger = log.New(f, "", log.LstdFlags)
		if loglevel >= 5 {
			connOpt = append(connOpt, jsonrpc2.LogMessages(config.Logger))
		}
	}

	if quiet && (logfile == "" || loglevel < 5) {
		connOpt = append(connOpt, jsonrpc2.LogMessages(log.New(ioutil.Discard, "", 0)))
	}

	handler := langserver.NewHandler(config)
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

func (c stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (c stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
