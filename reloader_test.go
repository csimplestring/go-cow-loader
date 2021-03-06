package gocowvalue

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type op int

func (o op) Type() string         { return "dummy" }
func (o op) Context() interface{} { return (int)(o) }

func Test_Queue(t *testing.T) {
	b := &queue{mutex: sync.Mutex{}}

	for i := 0; i < 100; i++ {
		go func() {
			b.add(op(1))
		}()
		go func() {
			ops := b.flush()
			if len(ops) < 0 {
				t.Error("fail")
			}
		}()
	}
}

type cowArray struct {
	arr []int
}

func (c *cowArray) Copy() Value {
	r := make([]int, len(c.arr))
	copy(r, c.arr)

	return &cowArray{
		arr: r,
	}
}

func (c *cowArray) Apply(ops []Op) error {
	for _, o := range ops {
		v := o.Context().(int)
		c.arr = append(c.arr, v)
	}
	return nil
}

func Test_Reloader(t *testing.T) {

	c := &cowArray{}
	r := New(c, 1)
	errChan := r.Err()

	go func() {
		for {
			select {
			case err := <-errChan:
				t.Error(err)
			}
		}
	}()

	for i := 0; i < 100; i++ {
		go func() {
			for {
				time.Sleep(1 * time.Second)
				r.Accept(op(1))
			}
		}()
	}

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		v := r.Reload().(*cowArray)
		fmt.Printf("%d \n", len(v.arr))
	}
}
