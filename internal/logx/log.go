package logx

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	}
	return "unknown"
}

type Logger struct {
	mu    sync.Mutex
	w     io.Writer
	level atomic.Int32
}

func New() *Logger { return NewWithWriter(os.Stderr) }

func NewWithWriter(w io.Writer) *Logger {
	l := &Logger{w: w}
	l.level.Store(int32(LevelInfo))
	return l
}

func (l *Logger) SetLevel(lv Level) {
	l.level.Store(int32(lv))
}

func (l *Logger) logf(lv Level, component, format string, args ...any) {
	if lv < Level(l.level.Load()) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = Redact(msg)
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s %s %s %s\n", ts, lv.String(), component, msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.w, line)
}

func (l *Logger) Info(component, format string, args ...any) { l.logf(LevelInfo, component, format, args...) }
func (l *Logger) Warn(component, format string, args ...any) { l.logf(LevelWarn, component, format, args...) }
func (l *Logger) Error(component, format string, args ...any) { l.logf(LevelError, component, format, args...) }
func (l *Logger) Debug(component, format string, args ...any) { l.logf(LevelDebug, component, format, args...) }

var Default = New()
