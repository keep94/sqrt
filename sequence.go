package sqrt

import (
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
}

// AsString returns all the digits in s as a string.
func AsString(s FiniteSequence) string {
	var sb strings.Builder
	for digit := range s.Values() {
		sb.WriteByte('0' + byte(digit))
	}
	return sb.String()
}

func opaqueSequence(s Sequence) Sequence {
	if _, ok := s.(*opqSequence); ok {
		return s
	}
	return &opqSequence{Sequence: s}
}

type opqSequence struct {
	Sequence
}

func (s *opqSequence) WithStart(start int) Sequence {
	result := s.Sequence.WithStart(start)
	if result == s.Sequence {
		return s
	}
	return opaqueSequence(result)
}
