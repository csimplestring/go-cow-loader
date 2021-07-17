# go-copyonwrite-loader
A simple tiny config loader by using copy-on-write idea.

## motivation 

Like the Java's CopyOnWriteArray, this tiny library implements the similar idea, but not only scoped to the array. Mutex can be used in a read/write concurrency environment, but not ideal in a write-less-read-more situation. 

This library is developed for reloading value in a write-less-read-more situation. To be more specific, the Loader accepts and buffers the modifications (called Op) on the value, periodically apply them on the value and reload it. The old value will be GC as long as it is not referenced any more. 

## example

``` Go
package  main

// op implements the Op interface
type op int
func (o op) Type() string         { return "dummy" }
func (o op) Context() interface{} { return (int)(o) }

// copy-on-write array implements the Value interface: Copy and Apply.
type cowArray struct {
	arr []int
}

// Copy() will deep-copy the value itself, this is important because if a pointer is still shared or shadow copied, the old value won't be GC.
func (c *cowArray) Copy() Value {
	r := make([]int, len(c.arr))
	copy(r, c.arr)

	return &cowArray{
		arr: r,
	}
}

// Apply() will apply the modification ops on the new value in the order as it is.
func (c *cowArray) Apply(ops []Op) error {
	for _, o := range ops {
		v := o.Context().(int)
		c.arr = append(c.arr, v)
	}
	return nil
}


func main() {

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
                // Accept is concurrent-safe, can be used by multiple go-routines
				r.Accept(op(1))
			}
		}()
	}

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
        // Reload is concurrent-safe, can be used by multiple go-routines
		v := r.Reload().(*cowArray)
		fmt.Printf("%d \n", len(v.arr))
	}

    // it should print like 
    // 101
    // 200
    // 300
    // 400
    // 500
    // 600
    // 700
    // 800
    // 900
}

```