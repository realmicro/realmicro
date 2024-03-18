package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	go_api "github.com/realmicro/realmicro/api/proto"
)

func TestRequestPayloadFromRequest(t *testing.T) {
	// our test event so that we can validate serializing / deserializing of true proto works
	protoEvent := go_api.Event{
		Name: "Test",
	}

	protoBytes, err := proto.Marshal(&protoEvent)
	if err != nil {
		t.Fatal("Failed to marshal proto", err)
	}

	jsonBytes, err := json.Marshal(protoEvent)
	if err != nil {
		t.Fatal("Failed to marshal proto to JSON ", err)
	}

	type jsonUrl struct {
		Key1 string `json:"key1"`
		Key2 string `json:"key2"`
		Name string `json:"name"`
	}
	jUrl := &jsonUrl{Key1: "val1", Key2: "val2", Name: "Test"}

	t.Run("extracting a json from a POST request with url params", func(t *testing.T) {
		r, err := http.NewRequest("POST", "http://localhost/my/path?key1=val1&key2=val2", bytes.NewReader(jsonBytes))
		if err != nil {
			t.Fatalf("Failed to created http.Request: %v", err)
		}

		extByte, err := requestPayload(r)
		if err != nil {
			t.Fatalf("Failed to extract payload from request: %v", err)
		}
		extJUrl := &jsonUrl{}
		if err := json.Unmarshal(extByte, extJUrl); err != nil {
			t.Fatalf("Failed to unmarshal payload from request: %v", err)
		}
		if !reflect.DeepEqual(extJUrl, jUrl) {
			t.Fatalf("Expected %v and %v to match", extJUrl, jUrl)
		}
	})

	t.Run("extracting a proto from a POST request", func(t *testing.T) {
		r, err := http.NewRequest("POST", "http://localhost/my/path", bytes.NewReader(protoBytes))
		if err != nil {
			t.Fatalf("Failed to created http.Request: %v", err)
		}

		extByte, err := requestPayload(r)
		if err != nil {
			t.Fatalf("Failed to extract payload from request: %v", err)
		}
		if string(extByte) != string(protoBytes) {
			t.Fatalf("Expected %v and %v to match", string(extByte), string(protoBytes))
		}
	})

	t.Run("extracting JSON from a POST request", func(t *testing.T) {
		r, err := http.NewRequest("POST", "http://localhost/my/path", bytes.NewReader(jsonBytes))
		if err != nil {
			t.Fatalf("Failed to created http.Request: %v", err)
		}

		extByte, err := requestPayload(r)
		if err != nil {
			t.Fatalf("Failed to extract payload from request: %v", err)
		}
		if string(extByte) != string(jsonBytes) {
			t.Fatalf("Expected %v and %v to match", string(extByte), string(jsonBytes))
		}
	})

	t.Run("extracting params from a GET request", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://localhost/my/path", nil)
		if err != nil {
			t.Fatalf("Failed to created http.Request: %v", err)
		}

		q := r.URL.Query()
		q.Add("name", "Test")
		r.URL.RawQuery = q.Encode()

		extByte, err := requestPayload(r)
		if err != nil {
			t.Fatalf("Failed to extract payload from request: %v", err)
		}
		if string(extByte) != string(jsonBytes) {
			t.Fatalf("Expected %v and %v to match", string(extByte), string(jsonBytes))
		}
	})

	t.Run("GET request with no params", func(t *testing.T) {
		r, err := http.NewRequest("GET", "http://localhost/my/path", nil)
		if err != nil {
			t.Fatalf("Failed to created http.Request: %v", err)
		}

		extByte, err := requestPayload(r)
		if err != nil {
			t.Fatalf("Failed to extract payload from request: %v", err)
		}
		if string(extByte) != "" {
			t.Fatalf("Expected %v and %v to match", string(extByte), "")
		}
	})
}
