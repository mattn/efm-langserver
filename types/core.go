package types

import "time"

const Wildcard = "="

// Config is
type Config struct {
	Version        int
	LogLevel       int
	Languages      *map[string][]Language
	RootMarkers    *[]string
	LintDebounce   time.Duration
	FormatDebounce time.Duration
}

// Language is
type Language struct {
	Prefix             string
	LintFormats        []string
	LintStdin          bool
	LintOffset         int
	LintOffsetColumns  int
	LintCommand        string
	LintIgnoreExitCode bool
	LintCategoryMap    map[string]string
	LintSource         string
	LintSeverity       DiagnosticSeverity
	LintWorkspace      bool
	LintAfterOpen      bool
	LintOnSave         bool
	FormatCommand      string
	FormatCanRange     bool
	FormatStdin        bool
	Env                []string
	RootMarkers        []string
	RequireMarker      bool
}

type EventType int

const (
	EventTypeChange EventType = iota
	EventTypeSave
	EventTypeOpen
)
