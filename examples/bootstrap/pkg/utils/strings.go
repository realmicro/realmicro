package utils

import (
	"os"
	"strings"
)

func TrimSpace(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

func EnvLookUpPrefixKey(prefix string) bool {
	hasPrefix := false
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 && strings.HasPrefix(pair[0], prefix) {
			hasPrefix = true
			break
		}
	}
	return hasPrefix
}

func LookupString(vs []string, v string) bool {
	for i := 0; i < len(vs); i++ {
		if v == vs[i] {
			return true
		}
	}
	return false
}
