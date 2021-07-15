package gocowvalue

import (
	"fmt"
	"testing"
	"time"
)

type op int

func (o op) Type() string         { return "dummy" }
func (o op) Context() interface{} { return (int)(o) }

func Test_OpBuffer(t *testing.T) {
	b := newOpBuffer()

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
	go r.Start()

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			r.Accept(op(1))
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Second)
			v := r.Reload().(*cowArray)
			fmt.Printf("%v \n", v.arr)
		}
	}()

	time.Sleep(5 * time.Second)
}
