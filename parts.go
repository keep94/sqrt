package sqrt

import (
	"context"
	"fmt"
	"iter"
	"math"
	"strings"
)

type mantissa struct {
	digits    *digitMemoizer
	maxDigits int
}

func newmantissa(digits func() int) mantissa {
	return mantissa{digits: newdigitMemoizer(digits), maxDigits: math.MaxInt}
}

func (m mantissa) At(posit int) int {
	if posit >= m.maxDigits {
		m.digits.At(m.maxDigits)
		return -1
	}
	return m.digits.At(posit)
}

func (m mantissa) ReverseScan(start int, yield func(index, value int) bool) {
	m.digits.ReverseScan(min(start, m.maxDigits), m.maxDigits, yield)
}

func (m mantissa) Scan(start int, yield func(index, value int) bool) {
	m.digits.Scan(min(start, m.maxDigits), m.maxDigits, yield)
}

func (m mantissa) ScanInRange(
	mantissaStart, start, end int, yield func(index, value int) bool) {
	m.digits.Scan(
		min(max(mantissaStart, start), m.maxDigits),
		min(end, m.maxDigits),
		yield)
}

func (m mantissa) ScanValues(start int, yield func(value int) bool) {
	m.digits.ScanValues(min(start, m.maxDigits), m.maxDigits, yield)
}

func (m mantissa) Values() iter.Seq[int] {
	return func(yield func(int) bool) {
		m.ScanValues(0, yield)
	}
}

func (m mantissa) PrimeToEnd(ctx context.Context) error {
	return m.digits.PrimeTo(ctx, m.maxDigits)
}

func (m mantissa) PrimeTo(ctx context.Context, upTo int) error {
	return m.digits.PrimeTo(ctx, min(upTo, m.maxDigits))
}

func (m mantissa) WithMaxDigits(maxDigits int) mantissa {
	if maxDigits <= 0 {
		return mantissa{}
	}
	result := m
	if maxDigits < result.maxDigits {
		result.maxDigits = maxDigits
	}
	return result
}

type sequencePart struct {
	mantissa mantissa
	start    int
}

func (s *sequencePart) All() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		s.mantissa.Scan(s.start, yield)
	}
}

func (s *sequencePart) AllInRange(start, end int) iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		s.mantissa.ScanInRange(s.start, start, end, yield)
	}
}

func (s *sequencePart) Values() iter.Seq[int] {
	return func(yield func(value int) bool) {
		s.mantissa.ScanValues(s.start, yield)
	}
}

func (s *sequencePart) PrimeToStart(ctx context.Context) error {
	return s.mantissa.PrimeTo(ctx, s.start)
}

func (s *sequencePart) primeToEnd(ctx context.Context) error {
	return s.mantissa.PrimeToEnd(ctx)
}

func (s *sequencePart) backward() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		s.mantissa.ReverseScan(s.start, yield)
	}
}

func (s *sequencePart) withStart(start int) sequencePart {
	result := *s
	if start > result.start {
		result.start = start
	}
	return result
}

func (s *sequencePart) withEnd(end int) sequencePart {
	result := *s
	result.mantissa = result.mantissa.WithMaxDigits(end)
	return result
}

type numberPart struct {
	mantissa mantissa
	exponent int
}

func (n *numberPart) All() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		n.mantissa.Scan(0, yield)
	}
}

func (n *numberPart) AllInRange(start, end int) iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		n.mantissa.ScanInRange(0, start, end, yield)
	}
}

func (n *numberPart) Values() iter.Seq[int] {
	return func(yield func(value int) bool) {
		n.mantissa.ScanValues(0, yield)
	}
}

func (n *numberPart) At(posit int) int {
	return n.mantissa.At(posit)
}

func (n *numberPart) Exponent() int {
	return n.exponent
}

func (n *numberPart) Format(state fmt.State, verb rune) {
	formatSpec, ok := newFormatSpec(state, verb, n.exponent)
	if !ok {
		fmt.Fprintf(state, "%%!%c(number=%s)", verb, n.String())
		return
	}
	formatSpec.PrintField(state, n)
}

func (n *numberPart) Exact() string {
	var builder strings.Builder
	fs := formatSpecForG(math.MaxInt, n.exponent, false)
	fs.PrintNumber(&builder, n)
	return builder.String()
}

func (n *numberPart) String() string {
	var builder strings.Builder
	fs := formatSpecForG(gPrecision, n.exponent, false)
	fs.PrintNumber(&builder, n)
	return builder.String()
}

func (n *numberPart) PrimeToStart(ctx context.Context) error {
	return nil
}

func (n *numberPart) IsZero() bool {
	return *n == numberPart{}
}

func (n *numberPart) primeToEnd(ctx context.Context) error {
	return n.mantissa.PrimeToEnd(ctx)
}

func (n *numberPart) backward() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		n.mantissa.ReverseScan(0, yield)
	}
}

func (n *numberPart) withExponent(e int) numberPart {
	result := *n
	if !result.IsZero() {
		result.exponent = e
	}
	return result
}

func (n *numberPart) withEnd(end int) numberPart {
	if end <= 0 {
		return numberPart{}
	}
	result := *n
	result.mantissa = result.mantissa.WithMaxDigits(end)
	return result
}
