package term

import (
	"io"
	"os"
	"sync"
)

var (
	isTerminalCache   = map[uintptr]bool{}
	isTerminalCacheMu sync.RWMutex
)

// IsTerminal checks whether the given writer is connected to a terminal.
//
// This function is thread-safe and efficient for repeated calls.
func IsTerminal(w io.Writer) bool {
	// The writer must be a file descriptor.
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()

	// Already cached.
	isTerminalCacheMu.RLock()
	v, exists := isTerminalCache[fd]
	isTerminalCacheMu.RUnlock()
	if exists {
		return v
	}

	// Check if the file descriptor is a terminal.
	info, err := f.Stat()
	isTTY := err == nil && (info.Mode()&os.ModeCharDevice) != 0

	// Cache the result.
	isTerminalCacheMu.Lock()
	isTerminalCache[fd] = isTTY
	isTerminalCacheMu.Unlock()

	return isTTY
}
