package strings

import "strconv"

func Int2String(in int64) string {
	return strconv.FormatInt(in, 10)
}

func String2Int(in string) (result int64) {
	result, _ = strconv.ParseInt(in, 10, 64)
	return
}

func Float2String(in float64) string {
	return strconv.FormatFloat(in, 'E', -1, 32)
}

func String2Float(in string) (result float64) {
	result, _ = strconv.ParseFloat(in, 64)
	return
}
