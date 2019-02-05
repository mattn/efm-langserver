package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/mattn/efm-langserver/langserver"
	"github.com/sourcegraph/jsonrpc2"
)

type efms []string

func (i *efms) String() string {
	return "efm list"
}

func (i *efms) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var efms efms
	var stdin bool
	var offset int
	flag.Var(&efms, "efm", "errorformat")
	flag.BoolVar(&stdin, "stdin", false, "use stdin")
	flag.IntVar(&offset, "offset", 0, "number offset")
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	log.Println("efm-langserver: reading on stdin, writing on stdout")

	exe := flag.Arg(0)
	args := flag.Args()[1:]

	if runtime.GOOS == "windows" && exe != "cmd" {
		found, err := exec.LookPath(exe)
		if err != nil || found == "" {
			exe = "cmd"
			args = append([]string{"/c", exe}, args...)
		}
	}
	handler := langserver.NewHandler(efms, stdin, offset, exe, args...)
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
