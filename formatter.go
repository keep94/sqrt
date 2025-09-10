package sqrt

import (
	"bufio"
	"io"
)

type formatter struct {
	writer          *bufio.Writer
	sigDigits       int // invariant sigDigits >= exponent
	exponent        int
	exactDigitCount bool
	index           int
}

func newFormatter(
	w io.Writer, sigDigits, exponent int, exactDigitCount bool) *formatter {
	if sigDigits < exponent {
		panic("sigDigits must be >= exponent")
	}
	return &formatter{
		writer:          bufio.NewWriter(w),
		sigDigits:       sigDigits,
		exponent:        exponent,
		exactDigitCount: exactDigitCount,
	}
}

func (f *formatter) CanConsume() bool {
	return f.index < f.sigDigits
}

func (f *formatter) Consume(digit int) {
	if !f.CanConsume() {
		return
	}
	f.add(digit)
}

func (f *formatter) Finish() {
	maxDigits := f.sigDigits
	if !f.exactDigitCount {
		maxDigits = f.exponent
	}
	for f.index < maxDigits {
		f.add(0)
	}
	// If we haven't written anything yet
	if f.index == 0 {
		count := -f.exponent
		if f.exactDigitCount {
			count = f.sigDigits - f.exponent
		}
		f.addLeadingZeros(count)
	}
	f.writer.Flush()
}

func (f *formatter) add(digit int) {
	if f.index == 0 && f.exponent <= 0 {
		f.addLeadingZeros(-f.exponent)
	}
	if f.index == f.exponent {
		f.writer.WriteByte('.')
	}
	f.writer.WriteByte('0' + byte(digit))
	f.index++
}

func (f *formatter) addLeadingZeros(count int) {
	f.writer.WriteByte('0')
	if count <= 0 {
		return
	}
	f.writer.WriteByte('.')
	for i := 0; i < count; i++ {
		f.writer.WriteByte('0')
	}
}
