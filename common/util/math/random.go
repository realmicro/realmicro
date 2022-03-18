package math

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandInt rand int between min and max
func RandInt(min, max int64) int64 {
	if min == max {
		return min
	} else if min > max {
		return 0
	}
	return min + rand.Int63n(max-min)
}
