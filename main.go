package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/konradmalik/efm-langserver/langserver"
)

const (
	name    = "efm-langserver"
	version = "0.0.54"
)

var revision = "HEAD"

func main() {
	var logfile string
	var loglevel int
	var showVersion bool
	var quiet bool

	flag.StringVar(&logfile, "logfile", "", "logfile")
	flag.IntVar(&loglevel, "loglevel", 1, "loglevel")
	flag.BoolVar(&showVersion, "v", false, "Print the version")
	flag.BoolVar(&quiet, "q", false, "Run quiet")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}

	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}

	var config *langserver.Config = langserver.NewConfig()
	config.LogLevel = loglevel

	if quiet {
		log.SetOutput(io.Discard)
	}

	log.Println("efm-langserver: reading on stdin, writing on stdout")

	var connOpt []jsonrpc2.ConnOpt

	config.LogFile = logfile
	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o660)
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
		connOpt = append(connOpt, jsonrpc2.LogMessages(log.New(io.Discard, "", 0)))
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
