package sqrt

import (
	"context"
	"iter"
	"strings"
)

// Sequence represents a sequence of digits of either finite or infinite
// length within the mantissa of a real number. Although they can start
// and optionally end anywhere within a mantissa, Sequences must be
// contiguous. That is they can have no gaps in the middle.
type Sequence interface {

	// All returns the 0 based position and value of each digit in this
	// Sequence from beginning to end.
	All() iter.Seq2[int, int]

	// AllInRange returns the 0 based position and value of each digit in
	// this sequence from position start up to but not including position
	// end.
	AllInRange(start, end int) iter.Seq2[int, int]

	// Values returns the value of each digit in this Sequence from
	// beginning to end.
	Values() iter.Seq[int]

	// WithStart returns a view of this Sequence that only has digits with
	// zero based positions greater than or equal to start.
	WithStart(start int) Sequence

	// WithEnd returns a view of this Sequence that only has digits with
	// zero based positions less than end.
	WithEnd(end int) FiniteSequence

	// PrimeToStart performs any necessary computations up front to ensure
	// that this sequence can be iterated over without any initial lag.
	PrimeToStart(ctx context.Context) error

	private()
}

// FiniteSequence represents a Sequence of finite length.
type FiniteSequence interface {
	Sequence

	// Backward returns the 0 based position and value of each digit in this
	// FiniteSequence from end to beginning.
	Backward() iter.Seq2[int, int]

	// FiniteWithStart works like WithStart except that it returns a
	// FiniteSequence.
	FiniteWithStart(start int) FiniteSequence

	// PrimeToEnd performs any necessary computations up front to ensure
	// that this sequence can be iterated over with Backward without any
	// initial lag.
	PrimeToEnd(ctx context.Context) error
}

// AsString returns all the digits in s as a string.
func AsString(s FiniteSequence) string {
	var sb strings.Builder
	for digit := range s.Values() {
		sb.WriteByte('0' + byte(digit))
	}
	return sb.String()
}

type sequence struct {
	sequencePart
}

func (s *sequence) WithStart(start int) Sequence {
	result := s.withStart(start)
	if result == s.sequencePart {
		return s
	}
	return &sequence{result}
}

func (s *sequence) WithEnd(end int) FiniteSequence {
	return &finiteSequence{s.withEnd(end)}
}

func (s *sequence) private() {
}

type finiteSequence struct {
	sequencePart
}

func (f *finiteSequence) WithStart(start int) Sequence {
	return f.FiniteWithStart(start)
}

func (f *finiteSequence) FiniteWithStart(start int) FiniteSequence {
	result := f.withStart(start)
	if result == f.sequencePart {
		return f
	}
	return &finiteSequence{result}
}

func (f *finiteSequence) WithEnd(end int) FiniteSequence {
	result := f.withEnd(end)
	if result == f.sequencePart {
		return f
	}
	return &finiteSequence{result}
}

func (f *finiteSequence) Backward() iter.Seq2[int, int] {
	return f.backward()
}

func (f *finiteSequence) PrimeToEnd(ctx context.Context) error {
	return f.primeToEnd(ctx)
}

func (f *finiteSequence) private() {
}
