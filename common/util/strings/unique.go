package strings

import (
	"strconv"
	"strings"
)

const (
	NumberBase    = "357182469"
	NumberDecimal = 9
	NumberPad     = "0"
)

func UniqueId(length int, uid int64) (result int64) {
	id := uid
	mod := int64(0)
	res := ""
	for id != 0 {
		mod = id % NumberDecimal
		id = id / NumberDecimal
		res += string(NumberBase[mod])
	}
	resLen := len(res)
	if resLen < length {
		res += NumberPad
		for i := 0; i < length-resLen-1; i++ {
			res += string(NumberBase[(int(uid)+i)%NumberDecimal])
		}
	}
	result, _ = strconv.ParseInt(res, 10, 64)
	return
}

func UniqueIdDecode(code string) int64 {
	res := int64(0)
	lenCode := len(code)
	baseArr := []byte(NumberBase)
	baseRev := make(map[byte]int)
	for k, v := range baseArr {
		baseRev[v] = k
	}
	isPad := strings.Index(code, NumberPad)
	if isPad != -1 {
		lenCode = isPad
	}
	r := 0
	for i := 0; i < lenCode; i++ {
		if string(code[i]) == NumberPad {
			continue
		}
		index, ok := baseRev[code[i]]
		if !ok {
			return 0
		}
		b := int64(1)
		for j := 0; j < r; j++ {
			b *= NumberDecimal
		}
		res += int64(index) * b
		r++
	}
	return res
}

const (
	BASE    = "E8S2DZX9WYLTN6BQF7CP5IK3MJUGR4HV"
	DECIMAL = 32
	PAD     = "A"
)

// UniqueStrId id to code
func UniqueStrId(length int, uid int64) string {
	id := uid
	mod := int64(0)
	res := ""
	for id != 0 {
		mod = id % DECIMAL
		id = id / DECIMAL
		res += string(BASE[mod])
	}
	resLen := len(res)
	if resLen < length {
		res += PAD
		for i := 0; i < length-resLen-1; i++ {
			res += string(BASE[(int(uid)+i)%DECIMAL])
		}
	}
	return res
}
