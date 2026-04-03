package sqrt

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrimeSequence(t *testing.T) {
	s := Sqrt(2).WithStart(1000)
	assert.NoError(t, s.PrimeToStart(context.Background()))
	for index := range s.All() {
		assert.Equal(t, 1000, index)
		break
	}
}

func TestPrimeSequenceCancel(t *testing.T) {
	var wg sync.WaitGroup
	s := Sqrt(2).WithStart(2000000)
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func(ctx context.Context) {
		assert.Equal(t, context.Canceled, s.PrimeToStart(ctx))
		wg.Done()
	}(ctx)
	cancel()
	wg.Wait()
}

func TestPrimeFiniteSequence(t *testing.T) {
	fs := Sqrt(2).WithStart(1000).WithEnd(2000)
	assert.NoError(t, fs.PrimeToEnd(context.Background()))
	for index := range fs.Backward() {
		assert.Equal(t, 1999, index)
		break
	}
}

func TestPrimeFiniteSequenceCancel(t *testing.T) {
	var wg sync.WaitGroup
	fs := Sqrt(2).WithStart(1000000).WithEnd(2000000)
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func(ctx context.Context) {
		assert.Equal(t, context.Canceled, fs.PrimeToEnd(ctx))
		wg.Done()
	}(ctx)
	cancel()
	wg.Wait()
}

func TestPrimeNumber(t *testing.T) {
	fn := Sqrt(2).WithEnd(2000000)
	assert.NoError(t, fn.PrimeToStart(context.Background()))
}

func TestPrimeFiniteNumber(t *testing.T) {
	var fn FiniteNumber
	assert.NoError(t, fn.PrimeToEnd(context.Background()))
}

func TestPrimeFiniteNumberCancel(t *testing.T) {
	var wg sync.WaitGroup
	fn := Sqrt(2).WithEnd(2000000)
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func(ctx context.Context) {
		assert.Equal(t, context.Canceled, fn.PrimeToEnd(ctx))
		wg.Done()
	}(ctx)
	cancel()
	wg.Wait()
}
