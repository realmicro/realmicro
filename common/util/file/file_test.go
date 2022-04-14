package file

import (
	"testing"
)

func TestExists(t *testing.T) {
	ok, err := PathExists("/")

	if ok {
		return
	}

	if !ok || err != nil {
		t.Fatalf("Test Exists fail, %s", err)
	}
}
