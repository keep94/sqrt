package sqrt

import (
	"math"
	"sync"
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

type memoizer struct {
	updateMu sync.Mutex
	iter     func() int
	readMu   sync.Mutex
	data     []int8
	done     bool
}

func newMemoizeSpec(iter func() int) numberSpec {
	return &memoizer{iter: iter}
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

func (m *memoizer) get() ([]int8, bool) {
	m.readMu.Lock()
	defer m.readMu.Unlock()
	return m.data, m.done
}

func (m *memoizer) put(data []int8, done bool) {
	m.readMu.Lock()
	defer m.readMu.Unlock()
	m.data, m.done = data, done
}

func getTargetLength(index int) int {
	chunkCount := index/kMemoizerChunkSize + 1

	// Have to prevent integer overflow in case index = math.MaxInt - 1
	if chunkCount > kMaxChunks {
		chunkCount = kMaxChunks
	}
	return kMemoizerChunkSize * chunkCount
}

func (m *memoizer) wait(index int) ([]int8, bool) {
	data, done := m.get()
	targetLength := getTargetLength(index)
	for !done && len(data) < targetLength {
		data, done = m.grow(targetLength)
	}
	return data, len(data) > index
}

func (m *memoizer) grow(targetLength int) ([]int8, bool) {
	m.updateMu.Lock()
	defer m.updateMu.Unlock()
	data, done := m.get()
	if !done && len(data) < targetLength {
		for range kMemoizerChunkSize {
			x := m.iter()
			if digitOutOfRange(x) {
				done = true
				break
			}
			data = append(data, int8(x))
		}
		m.put(data, done)
	}
	return data, done
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
