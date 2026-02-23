package sqrt

import (
	"math"
	"sync"
	"sync/atomic"
)

const (
	kMemoizerChunkSize = 100
	kMaxChunks         = math.MaxInt / kMemoizerChunkSize
)

type numberSpec interface {
	Scan(index, limit int, yield func(index, value int) bool)
	ScanValues(index, limit int, yield func(value int) bool)
	At(index int) int
	FirstN(n int) []int8
}

type context struct {
	active atomic.Int64
	closed atomic.Bool
}

func (c *context) Start() {
	c.active.Add(1)
}

func (c *context) End() {
	c.active.Add(-1)
}

func (c *context) NumActive() int64 {
	return c.active.Load()
}

func (c *context) Closed() bool {
	return c.closed.Load()
}

func (c *context) Close() {
	c.closed.Store(true)
}

type memoizer struct {
	iter            func() int
	ctxt            *context
	mu              sync.Mutex
	mustGrow        *sync.Cond
	updateAvailable *sync.Cond
	data            []int8
	maxLength       int
	done            bool
}

func newMemoizeSpec(iter func() int, ctxt *context) numberSpec {
	result := &memoizer{iter: iter, ctxt: ctxt}
	result.mustGrow = sync.NewCond(&result.mu)
	result.updateAvailable = sync.NewCond(&result.mu)
	ctxt.Start()
	go result.run()
	return result
}

func (m *memoizer) At(index int) int {
	if index < 0 {
		return -1
	}
	data, ok := m.wait(index)
	if !ok {
		return -1
	}
	return int(data[index])
}

func (m *memoizer) FirstN(n int) []int8 {
	if n <= 0 {
		return nil
	}
	data, _ := m.wait(n - 1)
	if len(data) > n {
		return data[:n]
	}
	return data
}

func (m *memoizer) Scan(index, limit int, yield func(index, value int) bool) {
	if index < 0 {
		panic("index must be non-negative")
	}
	data, ok := m.wait(index)
	for ok && index < limit {
		if !yield(index, int(data[index])) {
			return
		}
		index++
		if index == len(data) {
			data, ok = m.wait(index)
		}
	}
}

func (m *memoizer) ScanValues(index, limit int, yield func(value int) bool) {
	if index < 0 {
		panic("index must be non-negative")
	}
	data, ok := m.wait(index)
	for ok && index < limit {
		if !yield(int(data[index])) {
			return
		}
		index++
		if index == len(data) {
			data, ok = m.wait(index)
		}
	}
}

func (m *memoizer) wait(index int) ([]int8, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.done && m.maxLength <= index {
		chunkCount := index/kMemoizerChunkSize + 1

		// Have to prevent integer overflow in case index = math.MaxInt - 1
		if chunkCount > kMaxChunks {
			chunkCount = kMaxChunks
		}
		m.maxLength = kMemoizerChunkSize * chunkCount
		m.mustGrow.Signal()
	}
	for !m.done && len(m.data) <= index {
		m.updateAvailable.Wait()
	}
	return m.data, len(m.data) > index
}

func (m *memoizer) waitToGrow() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.data) >= m.maxLength {
		m.mustGrow.Wait()
	}
}

func (m *memoizer) setData(data []int8, done bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
	m.done = done
	m.updateAvailable.Broadcast()
}

func (m *memoizer) run() {
	defer m.ctxt.End()
	var data []int8
	for i := 0; i < kMaxChunks; i++ {
		m.waitToGrow()
		for j := 0; j < kMemoizerChunkSize; j++ {
			x := m.iter()
			if digitOutOfRange(x) {
				m.setData(data, true)
				return
			}
			data = append(data, int8(x))
		}
		if m.ctxt.Closed() {
			m.setData(data, true)
			return
		}
		m.setData(data, false)
	}
	m.setData(data, true)
}

type limitSpec struct {
	delegate numberSpec
	limit    int
}

func withLimit(spec numberSpec, limit int) numberSpec {
	if limit <= 0 || spec == nil {
		return nil
	}
	ls, ok := spec.(*limitSpec)
	if ok {
		if limit >= ls.limit {
			return spec
		}
		return &limitSpec{delegate: ls.delegate, limit: limit}
	}
	return &limitSpec{delegate: spec, limit: limit}
}

func (l *limitSpec) At(index int) int {
	if index >= l.limit {
		l.delegate.At(l.limit)
		return -1
	}
	return l.delegate.At(index)
}

func (l *limitSpec) Scan(index, limit int, yield func(index, value int) bool) {
	index = min(index, l.limit)
	limit = min(limit, l.limit)
	l.delegate.Scan(index, limit, yield)
}

func (l *limitSpec) ScanValues(index, limit int, yield func(value int) bool) {
	index = min(index, l.limit)
	limit = min(limit, l.limit)
	l.delegate.ScanValues(index, limit, yield)
}

func (l *limitSpec) FirstN(n int) []int8 {
	if n > l.limit {
		n = l.limit
	}
	return l.delegate.FirstN(n)
}

func digitOutOfRange(d int) bool {
	return d < 0 || d > 9
}
