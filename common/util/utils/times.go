package utils

import (
	"fmt"
	"time"

	mtime "github.com/realmicro/realmicro/common/util/time"
)

// An ElapsedTimer is a timer to track the elapsed time.
type ElapsedTimer struct {
	start time.Duration
}

// NewElapsedTimer returns an ElapsedTimer.
func NewElapsedTimer() *ElapsedTimer {
	return &ElapsedTimer{
		start: mtime.ReNow(),
	}
}

// Duration returns the elapsed time.
func (et *ElapsedTimer) Duration() time.Duration {
	return mtime.ReSince(et.start)
}

// Elapsed returns the string representation of elapsed time.
func (et *ElapsedTimer) Elapsed() string {
	return mtime.ReSince(et.start).String()
}

// ElapsedMs returns the elapsed time of string on milliseconds.
func (et *ElapsedTimer) ElapsedMs() string {
	return fmt.Sprintf("%.1fms", float32(mtime.ReSince(et.start))/float32(time.Millisecond))
}

// CurrentMicros returns the current microseconds.
func CurrentMicros() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// CurrentMillis returns the current milliseconds.
func CurrentMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
