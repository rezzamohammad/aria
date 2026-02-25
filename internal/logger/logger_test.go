package logger

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	l := New(100)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	if l.maxSize != 100 {
		t.Errorf("expected maxSize 100, got %d", l.maxSize)
	}
}

func TestNewDefaultMaxSize(t *testing.T) {
	l := New(0)
	if l.maxSize != 1000 {
		t.Errorf("expected default maxSize 1000, got %d", l.maxSize)
	}
}

func TestLog(t *testing.T) {
	l := New(100)
	l.Log(LevelInfo, "test", "hello world")

	entries := l.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Level != LevelInfo {
		t.Errorf("expected level info, got %s", entries[0].Level)
	}
	if entries[0].Source != "test" {
		t.Errorf("expected source 'test', got %s", entries[0].Source)
	}
	if entries[0].Message != "hello world" {
		t.Errorf("expected message 'hello world', got %s", entries[0].Message)
	}
}

func TestLogLevels(t *testing.T) {
	l := New(100)
	l.Info("src", "info msg")
	l.Warn("src", "warn msg")
	l.Error("src", "error msg")
	l.Debug("src", "debug msg")

	entries := l.Entries()
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	expected := []Level{LevelInfo, LevelWarn, LevelError, LevelDebug}
	for i, e := range entries {
		if e.Level != expected[i] {
			t.Errorf("entry %d: expected level %s, got %s", i, expected[i], e.Level)
		}
	}
}

func TestLogFormatting(t *testing.T) {
	l := New(100)
	l.Info("src", "count: %d, name: %s", 42, "aria")

	entries := l.Entries()
	if entries[0].Message != "count: 42, name: aria" {
		t.Errorf("expected formatted message, got %q", entries[0].Message)
	}
}

func TestMaxSizeEviction(t *testing.T) {
	l := New(5)
	for i := 0; i < 10; i++ {
		l.Info("src", "msg %d", i)
	}

	entries := l.Entries()
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries after eviction, got %d", len(entries))
	}
	// Should have entries 5-9
	if entries[0].Message != "msg 5" {
		t.Errorf("expected first entry 'msg 5', got %q", entries[0].Message)
	}
	if entries[4].Message != "msg 9" {
		t.Errorf("expected last entry 'msg 9', got %q", entries[4].Message)
	}
}

func TestRecent(t *testing.T) {
	l := New(100)
	for i := 0; i < 10; i++ {
		l.Info("src", "msg %d", i)
	}

	recent := l.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent entries, got %d", len(recent))
	}
	if recent[0].Message != "msg 7" {
		t.Errorf("expected 'msg 7', got %q", recent[0].Message)
	}
	if recent[2].Message != "msg 9" {
		t.Errorf("expected 'msg 9', got %q", recent[2].Message)
	}
}

func TestRecentMoreThanAvailable(t *testing.T) {
	l := New(100)
	l.Info("src", "only one")

	recent := l.Recent(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recent))
	}
}

func TestSubscribe(t *testing.T) {
	l := New(100)
	var received []Entry
	var mu sync.Mutex

	l.Subscribe(func(e Entry) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	l.Info("src", "test1")
	l.Warn("src", "test2")

	// Small delay for async delivery
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 received entries, got %d", len(received))
	}
	if received[0].Message != "test1" {
		t.Errorf("expected 'test1', got %q", received[0].Message)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	l := New(100)
	var count1, count2 int
	var mu sync.Mutex

	l.Subscribe(func(e Entry) {
		mu.Lock()
		count1++
		mu.Unlock()
	})
	l.Subscribe(func(e Entry) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	l.Info("src", "msg")
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Errorf("expected both subscribers to receive 1, got %d and %d", count1, count2)
	}
}

func TestConcurrentAccess(t *testing.T) {
	l := New(100)
	var wg sync.WaitGroup

	// Write from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				l.Info("goroutine", "msg %d-%d", n, j)
			}
		}(i)
	}

	// Read concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = l.Entries()
			_ = l.Recent(5)
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	entries := l.Entries()
	if len(entries) != 100 {
		t.Errorf("expected 100 entries, got %d", len(entries))
	}
}

func TestEntriesReturnsCopy(t *testing.T) {
	l := New(100)
	l.Info("src", "msg1")

	entries := l.Entries()
	entries[0].Message = "modified"

	// Original should be unchanged
	original := l.Entries()
	if original[0].Message != "msg1" {
		t.Error("Entries() should return a copy, not a reference")
	}
}
