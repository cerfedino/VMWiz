// Package logger does... logging.
//
// We have the concept of a "scope": it groups together log lines, e.g. for a big task or function. Each scope can have arbitrarily many sub-scopes, and has a unique ID through which we can browse its past logs.
//
// Scope "0" is the default catch-all scope. Each top-level scope stores its own logs plus all of its sub-scope logs into a single <rootid>.log file.
package logger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const RootScopeID = "0"

var Dir = "logs"

type ScopeStore interface {
	CreateLogScope(id string, parentID string, rootID string, label string) error
	FinishLogScope(id string, failed bool) error
	LogScopeRootID(id string) (string, error)
	LogScopeSubtreeIDs(id string) ([]string, error)
	LogScopeFinished(id string) (finished bool, failed bool, err error)
}

var store ScopeStore

func Init(dir string) error {
	regMu.Lock()
	for id, w := range writers {
		w.f.Close()
		delete(writers, id)
	}
	regMu.Unlock()
	Dir = dir
	return os.MkdirAll(dir, 0o755)
}

func SetStore(s ScopeStore) { store = s }

type Logger struct {
	scopeID string
	rootID  string
	w       *fileWriter
}

type logLine struct {
	Ts    string `json:"ts"`
	Level string `json:"level"`
	Scope string `json:"scope"`
	Msg   string `json:"msg"`
}

// write stamps the timestamp under the file lock so lines stay append-sorted.
func (l *Logger) write(level string, msg string) {
	if l.w == nil {
		fmt.Fprintln(os.Stderr, level, msg)
		return
	}
	l.w.mu.Lock()
	defer l.w.mu.Unlock()
	b, _ := json.Marshal(logLine{
		Ts:    time.Now().UTC().Format(time.RFC3339Nano),
		Level: level,
		Scope: l.scopeID,
		Msg:   msg,
	})
	l.w.f.Write(append(b, '\n'))
}

func (l *Logger) ScopeID() string { return l.scopeID }

func (l *Logger) Info(msg string)                { l.write("INFO", msg) }
func (l *Logger) Error(msg string)               { l.write("ERROR", msg) }
func (l *Logger) Infof(format string, a ...any)  { l.write("INFO", fmt.Sprintf(format, a...)) }
func (l *Logger) Errorf(format string, a ...any) { l.write("ERROR", fmt.Sprintf(format, a...)) }

func newLogger(scopeID string, rootID string) *Logger {
	w, _ := getWriter(rootID) // nil writer falls back to stderr in write
	return &Logger{scopeID: scopeID, rootID: rootID, w: w}
}

type fileWriter struct {
	mu sync.Mutex
	f  *os.File
}

var (
	regMu   sync.Mutex
	writers = map[string]*fileWriter{}
	idRe    = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func getWriter(rootID string) (*fileWriter, error) {
	if !idRe.MatchString(rootID) {
		return nil, fmt.Errorf("invalid scope id %q", rootID)
	}
	regMu.Lock()
	defer regMu.Unlock()
	if w, ok := writers[rootID]; ok {
		return w, nil
	}
	f, err := os.OpenFile(filepath.Join(Dir, rootID+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	w := &fileWriter{f: f}
	writers[rootID] = w
	return w, nil
}

func closeWriter(rootID string) {
	regMu.Lock()
	defer regMu.Unlock()
	if w, ok := writers[rootID]; ok {
		w.mu.Lock()
		w.f.Close()
		w.mu.Unlock()
		delete(writers, rootID)
	}
}

// StdWriter routes plain text lines (e.g. the standard log package) into the
// catch-all scope.
func StdWriter() io.Writer { return stdWriter{} }

type stdWriter struct{}

func (stdWriter) Write(p []byte) (int, error) {
	From(context.Background()).Info(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

type ctxKey struct{}

// Returns the logger carried by ctx, or the catch-all root logger.
func From(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return newLogger(RootScopeID, RootScopeID)
}

// Opens a child scope under the ctx logger and returns the updated ctx, the child logger, and a finish func that closes the scope.
func Nest(ctx context.Context, label string) (context.Context, *Logger, func(err error)) {
	parent := From(ctx)
	id, err := uuid.NewV7()
	if err != nil {
		return ctx, parent, func(error) {}
	}
	scopeID := id.String()
	rootID := parent.rootID
	if parent.scopeID == RootScopeID {
		rootID = scopeID
	}
	if store != nil {
		_ = store.CreateLogScope(scopeID, parent.scopeID, rootID, label)
	}
	child := newLogger(scopeID, rootID)
	finish := func(err error) {
		if err != nil {
			child.Error(err.Error())
		}
		if store != nil {
			_ = store.FinishLogScope(scopeID, err != nil)
		}
		if scopeID == rootID {
			closeWriter(rootID)
		}
	}
	return context.WithValue(ctx, ctxKey{}, child), child, finish
}

// Reports whether the on-disk log file for a root scope is present
func LogFileExists(rootID string) bool {
	_, err := os.Stat(filepath.Join(Dir, rootID+".log"))
	return err == nil
}

// ScopeFinished reports whether a scope has ended and whether it failed.
func ScopeFinished(id string) (finished bool, failed bool) {
	if store == nil {
		return false, false
	}
	finished, failed, err := store.LogScopeFinished(id)
	if err != nil {
		return false, false
	}
	return finished, failed
}

type Line struct {
	Ts    time.Time `json:"ts"`
	Level string    `json:"level"`
	Scope string    `json:"scope"`
	Msg   string    `json:"msg"`
}

// LogReader streams a scope's lines from a moving byte offset, so the same reader serves both history (first call) and live tailing (later calls).
type LogReader struct {
	path   string
	scopes []string // ignored when all is true
	all    bool
	offset int64
}

func NewLogReader(scopeID string, includeSubscopes bool) (*LogReader, error) {
	rootID, err := store.LogScopeRootID(scopeID)
	if err != nil {
		return nil, err
	}
	lr := &LogReader{path: filepath.Join(Dir, rootID+".log")}
	switch {
	case includeSubscopes && scopeID == rootID:
		// The file holds exactly this root's whole subtree, so stream every
		// line. Avoids snapshotting the subtree (sub-scopes created after the
		// stream opens would otherwise be filtered out).
		lr.all = true
	case includeSubscopes:
		lr.scopes, err = store.LogScopeSubtreeIDs(scopeID)
		if err != nil {
			return nil, err
		}
	default:
		lr.scopes = []string{scopeID}
	}
	return lr, nil
}

// Next returns the complete lines appended since the last call, advancing past them. A trailing partial line (mid-write) is left for the next call.
func (lr *LogReader) Next() ([]Line, error) {
	out := []Line{}
	f, err := os.Open(lr.path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	defer f.Close()
	if _, err := f.Seek(lr.offset, io.SeekStart); err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	for {
		raw, err := r.ReadBytes('\n')
		if len(raw) > 0 && raw[len(raw)-1] == '\n' {
			lr.offset += int64(len(raw))
			var l Line
			if json.Unmarshal(raw, &l) == nil &&
				(lr.all || slices.Contains(lr.scopes, l.Scope)) {
				out = append(out, l)
			}
		}
		if err != nil {
			return out, nil
		}
	}
}

// ReadLogs returns a scope's log lines, optionally including its sub-scopes, filtered to [from, to] (nil bounds are unbounded).
func ReadLogs(scopeID string, includeSubscopes bool, from *time.Time, to *time.Time) ([]Line, error) {
	lr, err := NewLogReader(scopeID, includeSubscopes)
	if err != nil {
		return nil, err
	}
	lines, err := lr.Next()
	if err != nil || (from == nil && to == nil) {
		return lines, err
	}
	out := []Line{}
	for _, l := range lines {
		if (from == nil || !l.Ts.Before(*from)) && (to == nil || !l.Ts.After(*to)) {
			out = append(out, l)
		}
	}
	return out, nil
}
