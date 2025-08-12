package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/konradmalik/efm-langserver/types"
	"github.com/reviewdog/errorformat"
)

var running = make(map[types.DocumentURI]context.CancelFunc)

type notifier interface {
	PublishDiagnostics(ctx context.Context, params types.PublishDiagnosticsParams)
	LogMessage(ctx context.Context, typ types.MessageType, message string)
}

func (h *LangHandler) ScheduleLinting(notifier notifier, uri types.DocumentURI, eventType types.EventType) {
	if h.lintTimer != nil {
		h.lintTimer.Reset(h.lintDebounce)
		if h.loglevel >= 4 {
			h.logger.Printf("lint debounced: %v", h.lintDebounce)
		}
		return
	}
	h.lintMu.Lock()
	h.lintTimer = time.AfterFunc(h.lintDebounce, func() {
		h.lintTimer = nil

		h.lintMu.Lock()
		cancel, ok := running[uri]
		if ok {
			cancel()
		}

		ctx, cancel := context.WithCancel(context.Background())
		running[uri] = cancel
		h.lintMu.Unlock()
		go h.runLintersPublishDiagnostics(ctx, notifier, uri, eventType)
	})
	h.lintMu.Unlock()
}

func (h *LangHandler) runLintersPublishDiagnostics(ctx context.Context, notifier notifier, uri types.DocumentURI, eventType types.EventType) {
	uriToDiagnostics, err := h.lintDocument(ctx, notifier, uri, eventType)
	if err != nil {
		h.logger.Println(err)
		return
	}

	for diagURI, diagnostics := range uriToDiagnostics {
		if diagURI == "file:" {
			diagURI = uri
		}
		version := 0
		if _, ok := h.files[uri]; ok {
			version = h.files[uri].Version
		}
		notifier.PublishDiagnostics(
			ctx,
			types.PublishDiagnosticsParams{
				URI:         diagURI,
				Diagnostics: diagnostics,
				Version:     version,
			})
	}
}

func (h *LangHandler) lintDocument(ctx context.Context, notifier notifier, uri types.DocumentURI, eventType types.EventType) (map[types.DocumentURI][]types.Diagnostic, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	fname, err := fromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)

	configs := lintConfigsForDocument(fname, f.LanguageID, h.configs, eventType)

	if len(configs) == 0 {
		if h.loglevel >= 2 {
			h.logger.Printf("lint for LanguageID not supported: %v", f.LanguageID)
		}
		return map[types.DocumentURI][]types.Diagnostic{}, nil
	}

	uriToDiagnostics := map[types.DocumentURI][]types.Diagnostic{
		uri: {},
	}
	publishedURIs := make(map[types.DocumentURI]struct{})
	for i, config := range configs {
		// To publish empty diagnostics when errors are fixed
		if config.LintWorkspace {
			for lastPublishedURI := range h.lastPublishedURIs[f.LanguageID] {
				if _, ok := uriToDiagnostics[lastPublishedURI]; !ok {
					uriToDiagnostics[lastPublishedURI] = []types.Diagnostic{}
				}
			}
		}

		if config.LintCommand == "" {
			continue
		}

		command := config.LintCommand
		if !config.LintStdin && !config.LintWorkspace && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		rootPath := h.findRootPath(fname, config)
		command = replaceCommandInputFilename(command, fname, rootPath)

		formats := config.LintFormats
		if len(formats) == 0 {
			formats = []string{"%f:%l:%m", "%f:%l:%c:%m"}
		}

		efms, err := errorformat.NewErrorformat(formats)
		if err != nil {
			return nil, fmt.Errorf("invalid error-format: %v", config.LintFormats)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/c", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}
		cmd.Dir = rootPath
		cmd.Env = append(os.Environ(), config.Env...)
		if config.LintStdin {
			cmd.Stdin = strings.NewReader(f.Text)
		}
		b, err := cmd.CombinedOutput()
		if err != nil {
			if succeeded(err) {
				return nil, nil
			}
		}
		// Most of lint tools exit with non-zero value. But some commands
		// return with zero value. We can not handle the output is real result
		// or output of usage. So efm-langserver ignore that command exiting
		// with zero-value. So if you want to handle the command which exit
		// with zero value, please specify lint-ignore-exit-code.
		if err == nil && !config.LintIgnoreExitCode {
			errorMessage := "command `" + command + "` exit with zero. probably you forgot to specify `lint-ignore-exit-code: true`."
			notifier.LogMessage(ctx, types.LogError, errorMessage)
			h.logger.Println(errorMessage)
			continue
		}
		if h.loglevel >= 3 {
			h.logger.Println(command+":", string(b))
		}
		var source *string
		if config.LintSource != "" {
			source = &configs[i].LintSource
		}

		var prefix string
		if config.Prefix != "" {
			prefix = fmt.Sprintf("[%s] ", config.Prefix)
		}

		scanner := efms.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			entry := scanner.Entry()
			if !entry.Valid {
				continue
			}
			if config.LintStdin && isFilename(entry.Filename) {
				entry.Filename = fname
				path, err := filepath.Abs(entry.Filename)
				if err != nil {
					continue
				}
				path = filepath.ToSlash(path)
				if runtime.GOOS == "windows" && !strings.EqualFold(path, fname) {
					continue
				} else if path != fname {
					continue
				}
			} else {
				entry.Filename = filepath.ToSlash(entry.Filename)
			}
			word := ""

			// entry.Col is expected to be one based, if the linter returns zero based we
			// have the ability to add an offset here.
			// We only add the offset if the linter reports entry.Col > 0 because 0 means the whole line
			if config.LintOffsetColumns > 0 && entry.Col > 0 {
				entry.Col = entry.Col + config.LintOffsetColumns
			}

			if entry.Lnum == 0 {
				entry.Lnum = 1 // entry.Lnum == 0 indicates the top line, set to 1 because it is subtracted later
			}

			if entry.Col == 0 {
				entry.Col = 1 // entry.Col == 0 indicates the whole line without column, set to 1 because it is subtracted later
			} else {
				word = f.wordAt(types.Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1})
			}

			diagURI := uri
			if entry.Filename != "" {
				if filepath.IsAbs(entry.Filename) {
					diagURI = toURI(entry.Filename)
				} else {
					diagURI = toURI(filepath.Join(rootPath, entry.Filename))
				}
			}
			if runtime.GOOS == "windows" {
				if !strings.EqualFold(string(diagURI), string(uri)) && !config.LintWorkspace {
					continue
				}
			} else {
				if diagURI != uri && !config.LintWorkspace {
					continue
				}
			}

			if config.LintWorkspace {
				publishedURIs[diagURI] = struct{}{}
			}
			uriToDiagnostics[diagURI] = append(uriToDiagnostics[diagURI], types.Diagnostic{
				Range: types.Range{
					Start: types.Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1},
					End:   types.Position{Line: entry.Lnum - 1 - config.LintOffset, Character: entry.Col - 1 + len([]rune(word))},
				},
				Code:     itoaPtrIfNotZero(entry.Nr),
				Message:  prefix + entry.Text,
				Severity: getSeverity(entry.Type, config.LintCategoryMap, config.LintSeverity),
				Source:   source,
			})
		}
	}

	// Update state here as no possibility of cancellation
	for _, config := range configs {
		if config.LintWorkspace {
			h.lastPublishedURIs[f.LanguageID] = publishedURIs
			break
		}
	}
	return uriToDiagnostics, nil
}
func getSeverity(typ rune, categoryMap map[string]string, defaultSeverity types.DiagnosticSeverity) types.DiagnosticSeverity {
	// we allow the config to provide a mapping between LSP types E,W,I,N and whatever categories the linter has
	if len(categoryMap) > 0 {
		typ = []rune(categoryMap[string(typ)])[0]
	}

	severity := types.Error
	if defaultSeverity != 0 {
		severity = defaultSeverity
	}

	switch typ {
	case 'E', 'e':
		severity = types.Error
	case 'W', 'w':
		severity = types.Warning
	case 'I', 'i':
		severity = types.Information
	case 'N', 'n':
		severity = types.Hint
	}
	return severity
}

func lintConfigsForDocument(fname, langId string, allConfigs map[string][]types.Language, eventType types.EventType) []types.Language {
	var configs []types.Language
	if cfgs, ok := allConfigs[langId]; ok {
		for _, cfg := range cfgs {
			// if we require markers and find that they dont exist we do not add the configuration
			if dir := matchRootPath(fname, cfg.RootMarkers); dir == "" && cfg.RequireMarker {
				continue
			}
			switch eventType {
			case types.EventTypeOpen:
				// if LintAfterOpen is not true, ignore didOpen
				if !cfg.LintAfterOpen {
					continue
				}
			case types.EventTypeChange:
				// if LintOnSave is true, ignore didChange
				if cfg.LintOnSave {
					continue
				}
			default:
			}
			if cfg.LintCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := allConfigs[types.Wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.LintCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	return configs
}
