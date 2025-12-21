package term

import (
	"bytes"
	"os"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	// Cleanup cache before tests.
	isTerminalCacheMu.Lock()
	isTerminalCache = map[uintptr]bool{}
	isTerminalCacheMu.Unlock()

	t.Run("not a file", func(t *testing.T) {
		buf := new(bytes.Buffer)
		if IsTerminal(buf) {
			t.Error("expected false for bytes.Buffer")
		}
	})

	t.Run("regular file is not tty", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "test_log")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		if IsTerminal(tmp) {
			t.Error("expected false for regular file")
		}

		// Check cache.
		if IsTerminal(tmp) {
			t.Error("expected false (cached) for regular file")
		}
	})

	t.Run("pipe is not tty", func(t *testing.T) {
		r, w, _ := os.Pipe()
		defer r.Close()
		defer w.Close()

		if IsTerminal(w) {
			t.Error("expected false for pipe")
		}
	})
}

func TestIsTerminalForcedCache(t *testing.T) {
	tmp, _ := os.CreateTemp("", "forced_test")
	defer os.Remove(tmp.Name())
	fd := tmp.Fd()

	// Manually inject into the cache.
	isTerminalCacheMu.Lock()
	isTerminalCache[fd] = true
	isTerminalCacheMu.Unlock()

	if !IsTerminal(tmp) {
		t.Error("expected true because it was manually injected in cache")
	}
}
