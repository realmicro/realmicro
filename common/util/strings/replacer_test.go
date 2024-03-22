package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplacer_Replace(t *testing.T) {
	mapping := map[string]string{
		"一二三四": "1234",
		"二三":   "23",
		"二":    "2",
	}
	assert.Equal(t, "零1234五", NewReplacer(mapping).Replace("零一二三四五"))
}

func TestReplacer_ReplaceLongestMatch(t *testing.T) {
	replacer := NewReplacer(map[string]string{
		"日本的首都": "东京",
		"日本":    "本日",
	})
	assert.Equal(t, "东京是东京", replacer.Replace("日本的首都是东京"))
}

func TestReplacer_ReplaceIndefinitely(t *testing.T) {
	mapping := map[string]string{
		"日本的首都": "东京",
		"东京":    "日本的首都",
	}
	assert.NotPanics(t, func() {
		NewReplacer(mapping).Replace("日本的首都是东京")
	})
}
