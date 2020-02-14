package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *langHandler) handleTextDocumentCodeAction(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params CodeActionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return h.codeAction(params.TextDocument.URI, &params)
}

func (h *langHandler) executeCommand(params *ExecuteCommandParams) (interface{}, error) {
	if len(params.Arguments) != 1 {
		return nil, fmt.Errorf("invalid command")
	}
	uri := fmt.Sprint(params.Arguments[0])
	fname, err := fromURI(uri)
	if err != nil {
		h.logger.Println("invalid uri")
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}

	var command *Command
	for i, v := range h.commands {
		if params.Command == v.Command {
			command = &h.commands[i]
			break
		}
	}
	if command == nil {
		return nil, fmt.Errorf("command not found: %v", params.Command)
	}

	var cmd *exec.Cmd
	var args []string
	if runtime.GOOS == "windows" {
		args = []string{"/c", command.Command}
		for _, v := range command.Arguments {
			arg := fmt.Sprint(v)
			if arg == "${INPUT}" {
				arg = fname
			}
			args = append(args, arg)
		}
		cmd = exec.Command("cmd", args...)
	} else {
		args = []string{"-c", command.Command}
		for _, v := range command.Arguments {
			arg := fmt.Sprint(v)
			if arg == "${INPUT}" {
				arg = fname
			}
			args = append(args, arg)
		}
		cmd = exec.Command("sh", args...)
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return string(b), nil
}

func (h *langHandler) codeAction(uri string, params *CodeActionParams) ([]Command, error) {
	commands := []Command{}
	for _, v := range h.commands {
		commands = append(commands, Command{
			Title:     v.Title,
			Command:   v.Command,
			Arguments: []interface{}{uri},
		})
	}
	return commands, nil
}
