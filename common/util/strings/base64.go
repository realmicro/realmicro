package strings

import "encoding/base64"

func Base64Encode(in string) (out string) {
	return base64.StdEncoding.EncodeToString([]byte(in))
}

func Base64Decode(in string) (out string) {
	outBytes, err := base64.StdEncoding.DecodeString(in)
	if err == nil {
		out = string(outBytes)
		return
	}
	return ""
}

func UrlEncode(in string) (out string) {
	return base64.URLEncoding.EncodeToString([]byte(in))
}

func UrlDecode(in string) (out string) {
	outBytes, err := base64.URLEncoding.DecodeString(in)
	if err == nil {
		out = string(outBytes)
		return
	}
	return ""
}
