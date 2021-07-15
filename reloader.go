package gocowvalue

import (
	"sync"
	"sync/atomic"
	"time"
)

type Op interface {
	Type() string
	Context() interface{}
}

type Value interface {
	Copy() Value
	Apply(ops []Op) error
}

type opBuffer struct {
	mutex sync.Mutex
	buf   []Op
	size  int
}

func newOpBuffer() *opBuffer {
	return &opBuffer{
		mutex: sync.Mutex{},
	}
}

func (o *opBuffer) add(op Op) {
	o.mutex.Lock()
	o.buf = append(o.buf, op)
	o.mutex.Unlock()
}

func (o *opBuffer) flush() []Op {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	r := o.buf
	o.buf = nil

	return r
}

type Reloader struct {
	ticker  *time.Ticker
	ops     *opBuffer
	atom    atomic.Value
	errChan chan error
}

func New(v Value, freshFreq int) *Reloader {
	r := &Reloader{
		ticker:  time.NewTicker(time.Second * time.Duration(freshFreq)),
		ops:     newOpBuffer(),
		atom:    atomic.Value{},
		errChan: make(chan error),
	}

	r.atom.Store(v)

	return r
}

func (r *Reloader) Reload() Value {
	return r.atom.Load().(Value)
}

func (r *Reloader) Accept(op Op) error {
	go r.ops.add(op)
	return nil
}

func (r *Reloader) Err() <-chan error {
	return r.errChan
}

func (r *Reloader) Start() {
	for {
		select {
		case <-r.ticker.C:
			ops := r.ops.flush()
			v := r.atom.Load().(Value)
			v2 := v.Copy()

			if err := v2.Apply(ops); err != nil {
				r.errChan <- err
			}

			r.atom.Store(v2)
		}
	}
}
