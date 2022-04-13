package strings

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// MD5Encrypt 给字符串生成md5，salt interface{} 加密的盐
func MD5Encrypt(str string, salt ...interface{}) (CryptStr string) {
	if l := len(salt); l > 0 {
		slice := make([]string, l+1)
		str = fmt.Sprintf(str+strings.Join(slice, "%v"), salt...)
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}
