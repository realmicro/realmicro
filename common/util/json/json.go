package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func Marshal(data interface{}) (result string) {
	if dataBytes, err := json.Marshal(data); err != nil {
		return ""
	} else {
		return string(dataBytes)
	}
}

// Unmarshal unmarshals data bytes into v.
func Unmarshal(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := unmarshalUseNumber(decoder, v); err != nil {
		return formatError(string(data), err)
	}

	return nil
}

// UnmarshalNE unmarshal with no error
func UnmarshalNE(data string, input interface{}) {
	_ = UnmarshalFromString(data, input)
}

// UnmarshalFromString unmarshals v from str.
func UnmarshalFromString(str string, v interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(str))
	if err := unmarshalUseNumber(decoder, v); err != nil {
		return formatError(str, err)
	}

	return nil
}

// UnmarshalFromReader unmarshals v from reader.
func UnmarshalFromReader(reader io.Reader, v interface{}) error {
	var buf strings.Builder
	teeReader := io.TeeReader(reader, &buf)
	decoder := json.NewDecoder(teeReader)
	if err := unmarshalUseNumber(decoder, v); err != nil {
		return formatError(buf.String(), err)
	}

	return nil
}

func unmarshalUseNumber(decoder *json.Decoder, v interface{}) error {
	decoder.UseNumber()
	return decoder.Decode(v)
}

func formatError(v string, err error) error {
	return fmt.Errorf("string: `%s`, error: `%w`", v, err)
}
