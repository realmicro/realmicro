package store

import (
	"fmt"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	m := NewMemoryStore()
	key := "myTest"
	rec := Record{
		Key:    key,
		Value:  []byte("myValue"),
		Expiry: 2 * time.Second,
	}

	err := m.Write(&rec)
	if err != nil {
		t.Errorf("Write Error: %v", err)
	}

	rec1, err := m.Read(key)
	if err != nil {
		t.Errorf("Read Error. Error: %v\n", err)
	}
	fmt.Println("record:", rec1[0].Key, string(rec1[0].Value))

	time.Sleep(3 * time.Second)

	_, err = m.Read(key)
	if err != nil {
		fmt.Println(err)
	}

	err = m.Write(&rec)
	if err != nil {
		t.Errorf("Write Error: %v", err)
	}

	err = m.Delete(key)
	if err != nil {
		t.Errorf("Delete error %v\n", err)
	}
	_, err = m.List()
	if err != nil {
		t.Errorf("list error %v\n", err)
	}
}
