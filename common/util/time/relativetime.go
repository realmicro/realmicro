package time

import "time"

// Use the long enough past time as start time, in case timex.ReNow() - lastTime equals 0.
var initTime = time.Now().AddDate(-1, -1, -1)

// ReNow returns a relative time duration since initTime, which is not important.
// The caller only needs to care about the relative value.
func ReNow() time.Duration {
	return time.Since(initTime)
}

// ReSince returns a diff since given d.
func ReSince(d time.Duration) time.Duration {
	return time.Since(initTime) - d
}

// ReTime returns current time, the same as time.ReNow().
func ReTime() time.Time {
	return initTime.Add(ReNow())
}
