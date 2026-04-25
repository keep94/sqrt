package sqrt

import (
	"context"
	"math"
	"sync"
)

const (
	kMemoizerChunkSize = 100
	kMaxChunks         = math.MaxInt / kMemoizerChunkSize
)

type digitMemoizer struct {
	updateMu sync.Mutex
	iter     func() int
	readMu   sync.Mutex
	data     []int8
	done     bool
}

func newdigitMemoizer(iter func() int) *digitMemoizer {
	return &digitMemoizer{iter: iter}
}

func (m *digitMemoizer) At(index int) int {
	if m == nil || index < 0 {
		return -1
	}
	data, ok := m.wait(index)
	if !ok {
		return -1
	}
	return int(data[index])
}

func (m *digitMemoizer) Scan(
	start, end int, yield func(index, value int) bool) {
	if start < 0 {
		panic("start must be non-negative")
	}
	if m == nil {
		return
	}
	data, ok := m.wait(start)
	for ok && start < end {
		if !yield(start, int(data[start])) {
			return
		}
		start++
		if start == len(data) {
			data, ok = m.wait(start)
		}
	}
}

func (m *digitMemoizer) ScanValues(
	start, end int, yield func(value int) bool) {
	if start < 0 {
		panic("start must be non-negative")
	}
	if m == nil {
		return
	}
	data, ok := m.wait(start)
	for ok && start < end {
		if !yield(int(data[start])) {
			return
		}
		start++
		if start == len(data) {
			data, ok = m.wait(start)
		}
	}
}

func (m *digitMemoizer) ReverseScan(
	start, end int, yield func(index, value int) bool) {
	if start < 0 {
		panic("start must be non-negative")
	}
	digits := m.firstN(end)
	for index := len(digits) - 1; index >= start; index-- {
		if !yield(index, int(digits[index])) {
			return
		}
	}
}

func (m *digitMemoizer) PrimeTo(ctx context.Context, upTo int) error {
	if m == nil || upTo <= 0 {
		return nil
	}
	data, done := m.get()
	targetLength := getTargetLength(upTo - 1)
	for !done && len(data) < targetLength {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data, done = m.grow(targetLength)
	}
	return nil
}

func (m *digitMemoizer) NumComputed() int {
	if m == nil {
		return 0
	}
	data, _ := m.get()
	return len(data)
}

func (m *digitMemoizer) firstN(n int) []int8 {
	if n <= 0 || m == nil {
		return nil
	}
	data, _ := m.wait(n - 1)
	if len(data) > n {
		return data[:n]
	}
	return data
}

func (m *digitMemoizer) get() ([]int8, bool) {
	m.readMu.Lock()
	defer m.readMu.Unlock()
	return m.data, m.done
}

func (m *digitMemoizer) put(data []int8, done bool) {
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

func (m *digitMemoizer) wait(index int) ([]int8, bool) {
	data, done := m.get()
	targetLength := getTargetLength(index)
	for !done && len(data) < targetLength {
		data, done = m.grow(targetLength)
	}
	return data, len(data) > index
}

func (m *digitMemoizer) grow(targetLength int) ([]int8, bool) {
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

func digitOutOfRange(d int) bool {
	return d < 0 || d > 9
}
