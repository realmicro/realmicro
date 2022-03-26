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

func SimilarText(first, second string, percent *float64) int {
	var similarText func(string, string, int, int) int
	similarText = func(str1, str2 string, len1, len2 int) int {
		var sum, max int
		pos1, pos2 := 0, 0

		// Find the longest segment of the same section in two strings
		for i := 0; i < len1; i++ {
			for j := 0; j < len2; j++ {
				for l := 0; (i+l < len1) && (j+l < len2) && (str1[i+l] == str2[j+l]); l++ {
					if l+1 > max {
						max = l + 1
						pos1 = i
						pos2 = j
					}
				}
			}
		}

		if sum = max; sum > 0 {
			if pos1 > 0 && pos2 > 0 {
				sum += similarText(str1, str2, pos1, pos2)
			}
			if (pos1+max < len1) && (pos2+max < len2) {
				s1 := []byte(str1)
				s2 := []byte(str2)
				sum += similarText(string(s1[pos1+max:]), string(s2[pos2+max:]), len1-pos1-max, len2-pos2-max)
			}
		}

		return sum
	}

	l1, l2 := len(first), len(second)
	if l1+l2 == 0 {
		return 0
	}
	sim := similarText(first, second, l1, l2)
	if percent != nil {
		*percent = float64(sim*200) / float64(l1+l2)
	}
	return sim
}
