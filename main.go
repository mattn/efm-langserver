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

	"github.com/konradmalik/efm-langserver/core"
	"github.com/konradmalik/efm-langserver/lsp"
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
	var usage bool

	flag.StringVar(&logfile, "logfile", "", "File to save logs into. If provided stderr won't be used anymore.")
	flag.IntVar(&loglevel, "loglevel", 1, "Set the log level. Max is 5, min is 0.")
	flag.BoolVar(&showVersion, "v", false, "Print the version")
	flag.BoolVar(&quiet, "q", false, "Run quiet")
	flag.BoolVar(&usage, "h", false, "Show help")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}

	if usage || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}

	config := core.NewConfig()
	config.LogLevel = loglevel

	if quiet {
		log.SetOutput(io.Discard)
	}

	log.Println("efm-langserver: reading on stdin, writing on stdout")

	var connOpt []jsonrpc2.ConnOpt

	var f *os.File
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()

	logger := createLogger(logfile)
	if !quiet && loglevel >= 5 {
		connOpt = append(connOpt, jsonrpc2.LogMessages(logger))
	} else {
		connOpt = append(connOpt, jsonrpc2.LogMessages(log.New(io.Discard, "", 0)))
	}

	internalHandler := core.NewHandler(logger, config)
	handler := lsp.NewHandler(internalHandler)
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.HandlerWithError(handler.Handle),
		connOpt...).DisconnectNotify()

	log.Println("efm-langserver: connections closed")
}

func createLogger(logfile string) *log.Logger {
	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o660)
		if err != nil {
			log.Fatal(err)
		}
		return log.New(f, "", log.LstdFlags)
	} else {
		return log.New(os.Stderr, "", log.LstdFlags)
	}
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
