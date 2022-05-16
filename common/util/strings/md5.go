package strings

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// MD5Encrypt string to md5，salt interface{}
func MD5Encrypt(str string, salt ...interface{}) (CryptStr string) {
	if l := len(salt); l > 0 {
		slice := make([]string, l+1)
		str = fmt.Sprintf(str+strings.Join(slice, "%v"), salt...)
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}
