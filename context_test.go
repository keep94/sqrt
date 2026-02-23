package sqrt

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	var ctxt Context
	n5 := ctxt.Sqrt(5)
	n7 := ctxt.Sqrt(7)
	n100489 := ctxt.Sqrt(100489)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		n5.At(2000000)
		wg.Done()
	}()
	assert.Equal(t, "2.645751311064590", n7.String())
	assert.Equal(t, "317", n100489.String())

	// n100489 was exhausted so its goroutine exited leaving the
	// goroutines for n5 and n7.
	assert.Equal(t, int64(2), ctxt.NumGoroutines())

	ctxt.Close()

	assert.Equal(t, int64(0), ctxt.NumGoroutines())
	assert.Equal(t, 0, ctxt.numSpecs())

	// Can't create numbers of a closed Context
	assert.Panics(t, func() {
		ctxt.Sqrt(13)
	})

	// Our long running goroutine should have ended.
	wg.Wait()

	// Test for idempotence
	ctxt.Close()
}
