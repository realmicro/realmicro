package file

import (
	"io"
	"os"
)

const bufSize = 1024

// PathExists returns file path if exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FirstLine returns the first line of the file.
func FirstLine(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return firstLine(file)
}

// LastLine returns the last line of the file.
func LastLine(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return lastLine(filename, file)
}

func firstLine(file *os.File) (string, error) {
	var first []byte
	var offset int64
	for {
		buf := make([]byte, bufSize)
		n, err := file.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return "", err
		}

		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				return string(append(first, buf[:i]...)), nil
			}
		}

		first = append(first, buf[:n]...)
		offset += bufSize
	}
}

func lastLine(filename string, file *os.File) (string, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return "", err
	}

	var last []byte
	offset := info.Size()
	for {
		offset -= bufSize
		if offset < 0 {
			offset = 0
		}
		buf := make([]byte, bufSize)
		n, err := file.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return "", err
		}

		if buf[n-1] == '\n' {
			buf = buf[:n-1]
			n--
		} else {
			buf = buf[:n]
		}
		for n--; n >= 0; n-- {
			if buf[n] == '\n' {
				return string(append(buf[n+1:], last...)), nil
			}
		}

		last = append(buf, last...)
	}
}
