package stream

import (
	"fmt"
	"testing"
	"time"
)

func TestStream(t *testing.T) {
	var ch = make(chan int)

	go func() {
		count := 0
		for {
			if count == 100 {
				break
			}
			ch <- count
			time.Sleep(time.Millisecond * 500)
			count++
		}
	}()

	From(func(source chan<- interface{}) {
		for c := range ch {
			source <- c
		}
	}).Walk(func(item interface{}, pipe chan<- interface{}) {
		count := item.(int)
		pipe <- count
	}).Filter(func(item interface{}) bool {
		itemInt := item.(int)
		if itemInt%2 == 0 {
			return true
		}
		return false
	}).ForEach(func(item interface{}) {
		fmt.Println(item)
	})
}
