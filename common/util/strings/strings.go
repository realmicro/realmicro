package strings

import (
	"fmt"
	"math/rand"
	"time"
	"unicode"
	"unicode/utf8"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func StarPhone(phone string) string {
	if len(phone) < 10 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func EndStarStr(str string) string {
	if len(str) == 0 {
		return "*"
	}
	count := utf8.RuneCountInString(str)
	if count == 0 || count == 1 {
		return "*"
	} else if count == 2 {
		return string([]rune(str)[:1]) + "*"
	} else {
		return string([]rune(str)[:count-2]) + "*"
	}
}

func ContainsDigits(str string) bool {
	for _, x := range []rune(str) {
		if unicode.IsDigit(x) {
			return true
		}
	}
	return false
}

func VerifyCode() string {
	return fmt.Sprintf("%06v", rand.Int31n(1000000))
}
