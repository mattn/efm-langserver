package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/konradmalik/efm-langserver/diff"
	"github.com/konradmalik/efm-langserver/types"
)

func (h *LangHandler) Formatting(uri types.DocumentURI, rng *types.Range, opt types.FormattingOptions) ([]types.TextEdit, error) {
	if h.formatTimer != nil {
		if h.loglevel >= 4 {
			h.logger.Printf("format debounced: %v", h.formatDebounce)
		}
		return []types.TextEdit{}, nil
	}

	h.formatMu.Lock()
	h.formatTimer = time.AfterFunc(h.formatDebounce, func() {
		h.formatMu.Lock()
		h.formatTimer = nil
		h.formatMu.Unlock()
	})
	h.formatMu.Unlock()
	return h.rangeFormatting(uri, rng, opt)
}

func (h *LangHandler) rangeFormatting(uri types.DocumentURI, rng *types.Range, options types.FormattingOptions) ([]types.TextEdit, error) {
	f, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found: %v", uri)
	}

	fname, err := fromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %v: %v", err, uri)
	}
	fname = filepath.ToSlash(fname)
	if runtime.GOOS == "windows" {
		fname = strings.ToLower(fname)
	}

	configs := formatConfigsForDocument(fname, f.LanguageID, h.configs)

	if len(configs) == 0 {
		if h.loglevel >= 2 {
			h.logger.Printf("format for LanguageID not supported: %v", f.LanguageID)
		}
		return nil, nil
	}

	originalText := f.Text
	text := originalText
	formatted := false

Configs:
	for _, config := range configs {
		if config.FormatCommand == "" {
			continue
		}

		// File options
		command := config.FormatCommand
		if !config.FormatStdin && !strings.Contains(command, "${INPUT}") {
			command = command + " ${INPUT}"
		}
		command = replaceCommandInputFilename(command, fname, h.RootPath)

		// Formatting Options
		for placeholder, value := range options {
			// {--flag:placeholder} => --flag <value>
			// {--flag=placeholder} => --flag=<value>
			// {--flag:!placeholder} => --flag if value is false
			re, err := regexp.Compile(fmt.Sprintf(`\${([^:|^}]+):%s}`, placeholder))
			re2, err2 := regexp.Compile(fmt.Sprintf(`\${([^=|^}]+)=%s}`, placeholder))
			nre, nerr := regexp.Compile(fmt.Sprintf(`\${([^:|^}]+):!%s}`, placeholder))
			nre2, nerr2 := regexp.Compile(fmt.Sprintf(`\${([^=|^}]+)=!%s}`, placeholder))
			if err != nil || err2 != nil || nerr != nil || nerr2 != nil {
				h.logger.Println(command+":", err)
				continue Configs
			}

			switch v := value.(type) {
			default:
				command = re.ReplaceAllString(command, fmt.Sprintf("$1 %v", v))
				command = re2.ReplaceAllString(command, fmt.Sprintf("$1=%v", v))
			case bool:
				const FLAG = "$1"
				if v {
					command = re.ReplaceAllString(command, FLAG)
					command = re2.ReplaceAllString(command, FLAG)
				} else {
					command = nre.ReplaceAllString(command, FLAG)
					command = nre2.ReplaceAllString(command, FLAG)
				}
			}
		}

		// Range Options
		if rng != nil {
			charStart := convertRowColToIndex(text, rng.Start.Line, rng.Start.Character)
			charEnd := convertRowColToIndex(text, rng.End.Line, rng.End.Character)

			rangeOptions := map[string]int{
				"charStart": charStart,
				"charEnd":   charEnd,
				"rowStart":  rng.Start.Line,
				"colStart":  rng.Start.Character,
				"rowEnd":    rng.End.Line,
				"colEnd":    rng.End.Character,
			}

			for placeholder, value := range rangeOptions {
				// {--flag:placeholder} => --flag <value>
				// {--flag=placeholder} => --flag=<value>
				re, err := regexp.Compile(fmt.Sprintf(`\${([^:|^}]+):%s}`, placeholder))
				re2, err2 := regexp.Compile(fmt.Sprintf(`\${([^=|^}]+)=%s}`, placeholder))
				if err != nil || err2 != nil {
					h.logger.Println(command+":", err)
					continue Configs
				}

				command = re.ReplaceAllString(command, fmt.Sprintf("$1 %d", value))
				command = re2.ReplaceAllString(command, fmt.Sprintf("$1=%d", value))
			}
		}

		// remove unfilled placeholders
		re := regexp.MustCompile(`\${[^}]*}`)
		command = re.ReplaceAllString(command, "")

		// Execute the command
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		cmd.Dir = h.findRootPath(fname, config)
		cmd.Env = append(os.Environ(), config.Env...)
		if config.FormatStdin {
			cmd.Stdin = strings.NewReader(text)
		}
		var buf bytes.Buffer
		cmd.Stderr = &buf
		b, err := cmd.Output()
		if err != nil {
			h.logger.Println(command+":", buf.String())
			continue
		}

		formatted = true

		if h.loglevel >= 3 {
			h.logger.Println(command+":", string(b))
		}
		text = strings.ReplaceAll(string(b), "\r", "")
	}

	if !formatted {
		return nil, fmt.Errorf("format for LanguageID not supported: %v", f.LanguageID)
	}

	if h.loglevel >= 3 {
		h.logger.Println("format succeeded")
	}
	return diff.ComputeEdits(uri, originalText, text), nil
}

func formatConfigsForDocument(fname, langId string, allConfigs map[string][]types.Language) []types.Language {
	var configs []types.Language
	if cfgs, ok := allConfigs[langId]; ok {
		for _, cfg := range cfgs {
			if cfg.FormatCommand != "" {
				if dir := matchRootPath(fname, cfg.RootMarkers); dir == "" && cfg.RequireMarker {
					continue
				}
				configs = append(configs, cfg)
			}
		}
	}
	if cfgs, ok := allConfigs[types.Wildcard]; ok {
		for _, cfg := range cfgs {
			if cfg.FormatCommand != "" {
				configs = append(configs, cfg)
			}
		}
	}
	return configs
}

func convertRowColToIndex(s string, row, col int) int {
	lines := strings.Split(s, "\n")

	if row < 0 {
		row = 0
	} else if row >= len(lines) {
		row = len(lines) - 1
	}

	if col < 0 {
		col = 0
	} else if col > len(lines[row]) {
		col = len(lines[row])
	}

	index := 0
	for i := 0; i < row; i++ {
		// Add the length of each line plus 1 for the newline character
		index += len(lines[i]) + 1
	}
	index += col

	return index
}
