package mapreduce

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var errDummy = errors.New("dummy")

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestFinish(t *testing.T) {
	defer goleak.VerifyNone(t)

	var total uint32
	err := Finish(func() error {
		atomic.AddUint32(&total, 2)
		return nil
	}, func() error {
		atomic.AddUint32(&total, 3)
		return nil
	}, func() error {
		atomic.AddUint32(&total, 5)
		return nil
	})

	assert.Equal(t, uint32(10), atomic.LoadUint32(&total))
	assert.Nil(t, err)
}

func TestFinishNone(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.Nil(t, Finish())
}

func TestFinishVoidNone(t *testing.T) {
	defer goleak.VerifyNone(t)

	FinishVoid()
}

func TestFinishErr(t *testing.T) {
	defer goleak.VerifyNone(t)

	var total uint32
	err := Finish(func() error {
		atomic.AddUint32(&total, 2)
		return nil
	}, func() error {
		atomic.AddUint32(&total, 3)
		return errDummy
	}, func() error {
		atomic.AddUint32(&total, 5)
		return nil
	})

	assert.Equal(t, errDummy, err)
}

func TestFinishVoid(t *testing.T) {
	defer goleak.VerifyNone(t)

	var total uint32
	FinishVoid(func() {
		atomic.AddUint32(&total, 2)
	}, func() {
		atomic.AddUint32(&total, 3)
	}, func() {
		atomic.AddUint32(&total, 5)
	})

	assert.Equal(t, uint32(10), atomic.LoadUint32(&total))
}

func TestForEach(t *testing.T) {
	const tasks = 1000

	t.Run("all", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		var count uint32
		ForEach(func(source chan<- interface{}) {
			for i := 0; i < tasks; i++ {
				source <- i
			}
		}, func(item interface{}) {
			atomic.AddUint32(&count, 1)
		}, WithWorkers(-1))

		assert.Equal(t, tasks, int(count))
	})

	t.Run("odd", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		var count uint32
		ForEach(func(source chan<- interface{}) {
			for i := 0; i < tasks; i++ {
				source <- i
			}
		}, func(item interface{}) {
			if item.(int)%2 == 0 {
				atomic.AddUint32(&count, 1)
			}
		})

		assert.Equal(t, tasks/2, int(count))
	})

	t.Run("all", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		assert.PanicsWithValue(t, "foo", func() {
			ForEach(func(source chan<- interface{}) {
				for i := 0; i < tasks; i++ {
					source <- i
				}
			}, func(item interface{}) {
				panic("foo")
			})
		})
	})
}

func TestGeneratePanic(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("all", func(t *testing.T) {
		assert.PanicsWithValue(t, "foo", func() {
			ForEach(func(source chan<- interface{}) {
				panic("foo")
			}, func(item interface{}) {
			})
		})
	})
}

func TestMapperPanic(t *testing.T) {
	defer goleak.VerifyNone(t)

	const tasks = 1000
	var run int32
	t.Run("all", func(t *testing.T) {
		assert.PanicsWithValue(t, "foo", func() {
			_, _ = MapReduce(func(source chan<- interface{}) {
				for i := 0; i < tasks; i++ {
					source <- i
				}
			}, func(item interface{}, writer Writer, cancel func(error)) {
				atomic.AddInt32(&run, 1)
				panic("foo")
			}, func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			})
		})
		assert.True(t, atomic.LoadInt32(&run) < tasks/2)
	})
}

func TestMapReduce(t *testing.T) {
	defer goleak.VerifyNone(t)

	tests := []struct {
		name        string
		mapper      MapperFunc
		reducer     ReducerFunc
		expectErr   error
		expectValue interface{}
	}{
		{
			name:        "simple",
			expectErr:   nil,
			expectValue: 30,
		},
		{
			name: "cancel with error",
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(errDummy)
				}
				writer.Write(v * v)
			},
			expectErr: errDummy,
		},
		{
			name: "cancel with nil",
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(nil)
				}
				writer.Write(v * v)
			},
			expectErr:   ErrCancelWithNil,
			expectValue: nil,
		},
		{
			name: "cancel with more",
			reducer: func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
				var result int
				for item := range pipe {
					result += item.(int)
					if result > 10 {
						cancel(errDummy)
					}
				}
				writer.Write(result)
			},
			expectErr: errDummy,
		},
	}

	t.Run("MapReduce", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.mapper == nil {
					test.mapper = func(item interface{}, writer Writer, cancel func(error)) {
						v := item.(int)
						writer.Write(v * v)
					}
				}
				if test.reducer == nil {
					test.reducer = func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
						var result int
						for item := range pipe {
							result += item.(int)
						}
						writer.Write(result)
					}
				}
				value, err := MapReduce(func(source chan<- interface{}) {
					for i := 1; i < 5; i++ {
						source <- i
					}
				}, test.mapper, test.reducer, WithWorkers(runtime.NumCPU()))

				assert.Equal(t, test.expectErr, err)
				assert.Equal(t, test.expectValue, value)
			})
		}
	})

	t.Run("MapReduce", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				if test.mapper == nil {
					test.mapper = func(item interface{}, writer Writer, cancel func(error)) {
						v := item.(int)
						writer.Write(v * v)
					}
				}
				if test.reducer == nil {
					test.reducer = func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
						var result int
						for item := range pipe {
							result += item.(int)
						}
						writer.Write(result)
					}
				}

				source := make(chan interface{})
				go func() {
					for i := 1; i < 5; i++ {
						source <- i
					}
					close(source)
				}()

				value, err := MapReduceChan(source, test.mapper, test.reducer, WithWorkers(-1))
				assert.Equal(t, test.expectErr, err)
				assert.Equal(t, test.expectValue, value)
			})
		}
	})
}

func TestMapReduceWithReduerWriteMoreThanOnce(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.Panics(t, func() {
		MapReduce(func(source chan<- interface{}) {
			for i := 0; i < 10; i++ {
				source <- i
			}
		}, func(item interface{}, writer Writer, cancel func(error)) {
			writer.Write(item)
		}, func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			drain(pipe)
			writer.Write("one")
			writer.Write("two")
		})
	})
}

func TestMapReduceVoid(t *testing.T) {
	defer goleak.VerifyNone(t)

	var value uint32
	tests := []struct {
		name        string
		mapper      MapperFunc
		reducer     VoidReducerFunc
		expectValue uint32
		expectErr   error
	}{
		{
			name:        "simple",
			expectValue: 30,
			expectErr:   nil,
		},
		{
			name: "cancel with error",
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(errDummy)
				}
				writer.Write(v * v)
			},
			expectErr: errDummy,
		},
		{
			name: "cancel with nil",
			mapper: func(item interface{}, writer Writer, cancel func(error)) {
				v := item.(int)
				if v%3 == 0 {
					cancel(nil)
				}
				writer.Write(v * v)
			},
			expectErr: ErrCancelWithNil,
		},
		{
			name: "cancel with more",
			reducer: func(pipe <-chan interface{}, cancel func(error)) {
				for item := range pipe {
					result := atomic.AddUint32(&value, uint32(item.(int)))
					if result > 10 {
						cancel(errDummy)
					}
				}
			},
			expectErr: errDummy,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			atomic.StoreUint32(&value, 0)

			if test.mapper == nil {
				test.mapper = func(item interface{}, writer Writer, cancel func(error)) {
					v := item.(int)
					writer.Write(v * v)
				}
			}
			if test.reducer == nil {
				test.reducer = func(pipe <-chan interface{}, cancel func(error)) {
					for item := range pipe {
						atomic.AddUint32(&value, uint32(item.(int)))
					}
				}
			}
			err := MapReduceVoid(func(source chan<- interface{}) {
				for i := 1; i < 5; i++ {
					source <- i
				}
			}, test.mapper, test.reducer)

			assert.Equal(t, test.expectErr, err)
			if err == nil {
				assert.Equal(t, test.expectValue, atomic.LoadUint32(&value))
			}
		})
	}
}

func TestMapReduceVoidWithDelay(t *testing.T) {
	defer goleak.VerifyNone(t)

	var result []int
	err := MapReduceVoid(func(source chan<- interface{}) {
		source <- 0
		source <- 1
	}, func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		if i == 0 {
			time.Sleep(time.Millisecond * 50)
		}
		writer.Write(i)
	}, func(pipe <-chan interface{}, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 1, result[0])
	assert.Equal(t, 0, result[1])
}

func TestMapReducePanic(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.Panics(t, func() {
		_, _ = MapReduce(func(source chan<- interface{}) {
			source <- 0
			source <- 1
		}, func(item interface{}, writer Writer, cancel func(error)) {
			i := item.(int)
			writer.Write(i)
		}, func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			for range pipe {
				panic("panic")
			}
		})
	})
}

func TestMapReducePanicOnce(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.Panics(t, func() {
		_, _ = MapReduce(func(source chan<- interface{}) {
			for i := 0; i < 100; i++ {
				source <- i
			}
		}, func(item interface{}, writer Writer, cancel func(error)) {
			i := item.(int)
			if i == 0 {
				panic("foo")
			}
			writer.Write(i)
		}, func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			for range pipe {
				panic("bar")
			}
		})
	})
}

func TestMapReducePanicBothMapperAndReducer(t *testing.T) {
	defer goleak.VerifyNone(t)

	assert.Panics(t, func() {
		_, _ = MapReduce(func(source chan<- interface{}) {
			source <- 0
			source <- 1
		}, func(item interface{}, writer Writer, cancel func(error)) {
			panic("foo")
		}, func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
			panic("bar")
		})
	})
}

func TestMapReduceVoidCancel(t *testing.T) {
	defer goleak.VerifyNone(t)

	var result []int
	err := MapReduceVoid(func(source chan<- interface{}) {
		source <- 0
		source <- 1
	}, func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		if i == 1 {
			cancel(errors.New("anything"))
		}
		writer.Write(i)
	}, func(pipe <-chan interface{}, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
	})
	assert.NotNil(t, err)
	assert.Equal(t, "anything", err.Error())
}

func TestMapReduceVoidCancelWithRemains(t *testing.T) {
	defer goleak.VerifyNone(t)

	var done int32
	var result []int
	err := MapReduceVoid(func(source chan<- interface{}) {
		for i := 0; i < defaultWorkers*2; i++ {
			source <- i
		}
		atomic.AddInt32(&done, 1)
	}, func(item interface{}, writer Writer, cancel func(error)) {
		i := item.(int)
		if i == defaultWorkers/2 {
			cancel(errors.New("anything"))
		}
		writer.Write(i)
	}, func(pipe <-chan interface{}, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
	})
	assert.NotNil(t, err)
	assert.Equal(t, "anything", err.Error())
	assert.Equal(t, int32(1), done)
}

func TestMapReduceWithoutReducerWrite(t *testing.T) {
	defer goleak.VerifyNone(t)

	uids := []int{1, 2, 3}
	res, err := MapReduce(func(source chan<- interface{}) {
		for _, uid := range uids {
			source <- uid
		}
	}, func(item interface{}, writer Writer, cancel func(error)) {
		writer.Write(item)
	}, func(pipe <-chan interface{}, writer Writer, cancel func(error)) {
		drain(pipe)
		// not calling writer.Write(...), should not panic
	})
	assert.Equal(t, ErrReduceNoOutput, err)
	assert.Nil(t, res)
}

func TestMapReduceVoidPanicInReducer(t *testing.T) {
	defer goleak.VerifyNone(t)

	const message = "foo"
	assert.Panics(t, func() {
		var done int32
		_ = MapReduceVoid(func(source chan<- interface{}) {
			for i := 0; i < defaultWorkers*2; i++ {
				source <- i
			}
			atomic.AddInt32(&done, 1)
		}, func(item interface{}, writer Writer, cancel func(error)) {
			i := item.(int)
			writer.Write(i)
		}, func(pipe <-chan interface{}, cancel func(error)) {
			panic(message)
		}, WithWorkers(1))
	})
}

func TestForEachWithContext(t *testing.T) {
	defer goleak.VerifyNone(t)

	var done int32
	ctx, cancel := context.WithCancel(context.Background())
	ForEach(func(source chan<- interface{}) {
		for i := 0; i < defaultWorkers*2; i++ {
			source <- i
		}
		atomic.AddInt32(&done, 1)
	}, func(item interface{}) {
		i := item.(int)
		if i == defaultWorkers/2 {
			cancel()
		}
	}, WithContext(ctx))
}

func TestMapReduceWithContext(t *testing.T) {
	defer goleak.VerifyNone(t)

	var done int32
	var result []int
	ctx, cancel := context.WithCancel(context.Background())
	err := MapReduceVoid(func(source chan<- interface{}) {
		for i := 0; i < defaultWorkers*2; i++ {
			source <- i
		}
		atomic.AddInt32(&done, 1)
	}, func(item interface{}, writer Writer, c func(error)) {
		i := item.(int)
		if i == defaultWorkers/2 {
			cancel()
		}
		writer.Write(i)
	}, func(pipe <-chan interface{}, cancel func(error)) {
		for item := range pipe {
			i := item.(int)
			result = append(result, i)
		}
	}, WithContext(ctx))
	assert.NotNil(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func BenchmarkMapReduce(b *testing.B) {
	b.ReportAllocs()

	mapper := func(v interface{}, writer Writer, cancel func(error)) {
		writer.Write(v.(int64) * v.(int64))
	}
	reducer := func(input <-chan interface{}, writer Writer, cancel func(error)) {
		var result int64
		for v := range input {
			result += v.(int64)
		}
		writer.Write(result)
	}

	for i := 0; i < b.N; i++ {
		MapReduce(func(input chan<- interface{}) {
			for j := 0; j < 2; j++ {
				input <- int64(j)
			}
		}, mapper, reducer)
	}
}
