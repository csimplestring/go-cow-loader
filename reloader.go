package gocowvalue

import (
	"sync"
	"sync/atomic"
	"time"
)

// Op defines the modification that will be applied on Value.
type Op interface {
	Type() string
	Context() interface{}
}

// Value provides 2 functions in order to fulfill the copy-on-write:
// Copy() will deep-copy the value itself, this is important because if a pointer is still shared or shadow
// copied, the old value won't be GC.
// Apply() will apply the modification ops, whether in a sequential order or arbitrary order, depends on the
// value implementation. This reload can not guarantee anything.
type Value interface {
	Copy() Value
	Apply(ops []Op) error
}

// queue is a synchronised blocking slice.
type queue struct {
	mutex sync.Mutex
	buf   []Op
	size  int
}

// add adds an op to q.
func (q *queue) add(op Op) {
	q.mutex.Lock()
	q.buf = append(q.buf, op)
	q.mutex.Unlock()
}

// flush cleans up internal buffer and return it.
func (q *queue) flush() []Op {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	r := q.buf
	q.buf = nil

	return r
}

// Reloader reloads a value periodically by using a copy-on-write mechanism.
type Reloader struct {
	ticker  *time.Ticker
	ops     *queue
	atom    atomic.Value
	errChan chan error
}

// New creates a new loader for v, and periodically refresh it.
func New(v Value, freshFreq int) *Reloader {
	r := &Reloader{
		ticker:  time.NewTicker(time.Second * time.Duration(freshFreq)),
		ops:     &queue{mutex: sync.Mutex{}},
		atom:    atomic.Value{},
		errChan: make(chan error),
	}

	r.atom.Store(v)

	go r.run()

	return r
}

// Reload returns the latest snapshot of value. The loaded value is not guaranteed to contain all the modifications,
// see Accept(op Op).
func (r *Reloader) Reload() Value {
	return r.atom.Load().(Value)
}

// Accept appends the modification op into a queue, note that all the ops are not applied immediately
// but in a batch fashion once the ticker is triggered.
func (r *Reloader) Accept(op Op) error {
	go r.ops.add(op)
	return nil
}

// Err returns an error channel.
func (r *Reloader) Err() <-chan error {
	return r.errChan
}

// run starts an for-loop to periodically do the copy-on-write thing:
// the ticker is triggered, then copy the old value, apply all the buffered modifications to the new value
// store the new value.
func (r *Reloader) run() {
	for {
		select {
		case <-r.ticker.C:
			ops := r.ops.flush()
			v := r.atom.Load().(Value)
			// what if copy times out?
			v2 := v.Copy()

			if err := v2.Apply(ops); err != nil {
				r.errChan <- err
			}

			r.atom.Store(v2)
		}
	}
}
