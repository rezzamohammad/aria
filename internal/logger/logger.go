package logger

import (
	"fmt"
	"sync"
	"time"
)

// Level represents log severity.
type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelDebug Level = "debug"
)

// Entry represents a single log event.
type Entry struct {
	Timestamp time.Time
	Level     Level
	Source    string // agent ID, "orchestrator", "system"
	Message   string
}

// Listener is a callback that receives log entries.
type Listener func(Entry)

// Logger is a thread-safe event logger with subscriber support.
type Logger struct {
	mu        sync.RWMutex
	entries   []Entry
	listeners []Listener
	maxSize   int
}

// New creates a new Logger.
func New(maxSize int) *Logger {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &Logger{
		entries: make([]Entry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Subscribe registers a listener that will be called for every new entry.
func (l *Logger) Subscribe(fn Listener) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.listeners = append(l.listeners, fn)
}

// Log adds an entry and notifies all listeners.
func (l *Logger) Log(level Level, source, message string) {
	entry := Entry{
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
	}

	l.mu.Lock()
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
	listeners := make([]Listener, len(l.listeners))
	copy(listeners, l.listeners)
	l.mu.Unlock()

	for _, fn := range listeners {
		fn(entry)
	}
}

// Info logs an info message.
func (l *Logger) Info(source, format string, args ...interface{}) {
	l.Log(LevelInfo, source, fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *Logger) Warn(source, format string, args ...interface{}) {
	l.Log(LevelWarn, source, fmt.Sprintf(format, args...))
}

// Error logs an error message.
func (l *Logger) Error(source, format string, args ...interface{}) {
	l.Log(LevelError, source, fmt.Sprintf(format, args...))
}

// Debug logs a debug message.
func (l *Logger) Debug(source, format string, args ...interface{}) {
	l.Log(LevelDebug, source, fmt.Sprintf(format, args...))
}

// Entries returns a copy of all log entries.
func (l *Logger) Entries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Entry, len(l.entries))
	copy(result, l.entries)
	return result
}

// Recent returns the last N entries.
func (l *Logger) Recent(n int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if n >= len(l.entries) {
		result := make([]Entry, len(l.entries))
		copy(result, l.entries)
		return result
	}
	start := len(l.entries) - n
	result := make([]Entry, n)
	copy(result, l.entries[start:])
	return result
}
