// Package sqrt computes square roots and cube roots to arbitrary precision.
//
// Number is the main type in this package. It represents a lazily evaluated
// non negative real number that generally has an infinite number of digits,
// but a Number can also have a finite number of digits.
//
// A FiniteNumber works like Number except that it always has a finite
// number of digits. A *FiniteNumber can be used anywhere a Number type
// is expected but not the other way around.
//
// A Sequence is a view of a contiguous subset of digits of a Number.
// For example, A Sequence could represent everything past the 1000th digit
// of the square root of 3. Because Sequences are views, they are cheap to
// create. Note that Number and *FiniteNumber can be used anywhere a Sequence
// type is expected. A Sequence can be either infinite or finite in length.
//
// A FiniteSequence works like Sequence except unlike Sequence, a
// FiniteSequence is always finite in length. A FiniteSequence can be used
// anywhere a Sequence is expected, and a *FiniteNumber can be used anywhere
// a FiniteSequence is expected. However, a Number or Sequence cannot be
// used where a FiniteSequence is expected because they can have an infinite
// number of digits. A FiniteSequence must have a finite number of digits.
package sqrt

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"math/big"
	"strings"
	"sync"
)

const (
	fPrecision = 6
	gPrecision = 16
)

var (
	zeroNumber           = &FiniteNumber{}
	globalNumSpecContext = &context{}
	nilContext           *Context
)

var (
	_ FiniteSequence = zeroNumber
	_ Number         = zeroNumber
)

// Use a Context instance to create Number instances when you want to
// free resources used by those Number instances before the program ends.
// Closing a Context frees all resources used by the Number instances
// it created. Once a Context is closed, it panics if used to create
// Numbers. The zero value of Context is ready to use.
//
// Never copy a Context instance.
//
// Prefer using the methods of Context to create Numbers over the free
// standing factory functions. The resources used by Numbers instances
// created with free standing factory functions such as sqrt.Sqrt never
// get freed.
type Context struct {
	mu    sync.Mutex
	ctxt  context
	specs []numberSpec
}

// Sqrt returns the square root of radican. Sqrt panics if radican is
// negative.
func (c *Context) Sqrt(radican int64) Number {
	return c.nRootFrac(big.NewInt(radican), one, newSqrtManager)
}

// SqrtRat returns the square root of num / denom. denom must be positive,
// and num must be non-negative or else SqrtRat panics.
func (c *Context) SqrtRat(num, denom int64) Number {
	return c.nRootFrac(big.NewInt(num), big.NewInt(denom), newSqrtManager)
}

// SqrtBigInt returns the square root of radican. SqrtBigInt panics if
// radican is negative.
func (c *Context) SqrtBigInt(radican *big.Int) Number {
	return c.nRootFrac(radican, one, newSqrtManager)
}

// SqrtBigRat returns the square root of radican. The denominator of radican
// must be positive, and the numerator must be non-negative or else SqrtBigRat
// panics.
func (c *Context) SqrtBigRat(radican *big.Rat) Number {
	return c.nRootFrac(radican.Num(), radican.Denom(), newSqrtManager)
}

// CubeRoot returns the cube root of radican. CubeRoot panics if radican is
// negative as Number can only hold positive results.
func (c *Context) CubeRoot(radican int64) Number {
	return c.nRootFrac(big.NewInt(radican), one, newCubeRootManager)
}

// CubeRootRat returns the cube root of num / denom. Because Number can only
// hold positive results, denom must be positive, and num must be non-negative
// or else CubeRootRat panics.
func (c *Context) CubeRootRat(num, denom int64) Number {
	return c.nRootFrac(big.NewInt(num), big.NewInt(denom), newCubeRootManager)
}

// CubeRootBigInt returns the cube root of radican. CubeRootBigInt panics if
// radican is negative as Number can only hold positive results.
func (c *Context) CubeRootBigInt(radican *big.Int) Number {
	return c.nRootFrac(radican, one, newCubeRootManager)
}

// CubeRootBigRat returns the cube root of radican. Because Number can only
// hold positive results, the denominator of radican must be positive, and the
// numerator must be non-negative or else CubeRootBigRat panics.
func (c *Context) CubeRootBigRat(radican *big.Rat) Number {
	return c.nRootFrac(radican.Num(), radican.Denom(), newCubeRootManager)
}

// NewNumberForTesting creates an arbitrary Number for testing. fixed are
// digits between 0 and 9 representing the non repeating digits that come
// immediately after the decimal place of the mantissa. repeating are digits
// between 0 and 9 representing the repeating digits that follow the non
// repeating digits of the mantissa. exp is the exponent part of the
// returned Number. NewNumberForTesting returns an error if fixed or
// repeating contain values not between 0 and 9, or if the first digit of
// the mantissa would be zero since mantissas must be between 0.1 inclusive
// and 1.0 exclusive.
func (c *Context) NewNumberForTesting(fixed, repeating []int, exp int) (Number, error) {
	if len(fixed) == 0 && len(repeating) == 0 {
		return zeroNumber, nil
	}
	if !validDigits(fixed) || !validDigits(repeating) {
		return nil, errors.New("NewNumberForTesting: digits must be between 0 and 9")
	}
	gen := newRepeatingGenerator(fixed, repeating, exp)
	digits, _ := gen.Generate()
	if digits() == 0 {
		return nil, errors.New("NewNumberForTesting: leading zeros not allowed in digits")
	}
	if len(repeating) == 0 {
		return c.newFiniteNumber(gen.Generate()), nil
	}
	return c.newNumber(gen.Generate()), nil
}

// NewNumber returns a new Number based on g. Although g is expected to
// follow the contract of Generator, if g yields mantissa digits outside the
// range of 0 and 9, NewNumber regards that as a signal that there are no
// more mantissa digits. Also if g happens to yield 0 as the first digit
// of the mantissa, NewNumber will return zero.
func (c *Context) NewNumber(g Generator) Number {
	digits, exp := g.Generate()
	first := digits()
	if first == 0 || digitOutOfRange(first) {
		return zeroNumber
	}
	return c.newNumber(firstAndThen(first, digits), exp)
}

// NewFiniteNumber works like NewNumberForTesting except that it
// returns a *FiniteNumber instead of a Number. Note that there is no
// repeating parameter because FiniteNumbers have a finite number of
// digits.
func (c *Context) NewFiniteNumber(fixed []int, exponent int) (*FiniteNumber, error) {
	result, err := c.NewNumberForTesting(fixed, nil, exponent)
	if err != nil {
		return nil, err
	}
	return result.(*FiniteNumber), nil
}

// Close closes this Context freeing all resources used by the Number
// instances it created.
func (c *Context) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctxt.Close()
	for _, spec := range c.specs {
		spec.FirstN(math.MaxInt)
	}
	c.specs = nil
}

// NumGoroutines returns the number of active goroutines in use to generate
// digits for Numbers this Context created. Returns 0 after Close() is called.
func (c *Context) NumGoroutines() int64 {
	if c == nil {
		return globalNumSpecContext.NumActive()
	}
	return c.ctxt.NumActive()
}

func (c *Context) numSpecs() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.specs)
}

func (c *Context) nRootFrac(
	num, denom *big.Int, newManager func() rootManager) Number {
	checkNumDenom(num, denom)
	if num.Sign() == 0 {
		return zeroNumber
	}
	return c.newNumber(newNRootGenerator(num, denom, newManager).Generate())
}

// newNumber returns a new number. The first digit that digits generates
// must be between 1 and 9.
func (c *Context) newNumber(digits func() int, exp int) Number {
	return opaqueNumber(c.newFiniteNumber(digits, exp))
}

func (c *Context) newFiniteNumber(digits func() int, exp int) *FiniteNumber {
	mantissa := mantissa{spec: c.newMemoizeSpec(digits)}
	return &FiniteNumber{exponent: exp, mantissa: mantissa}
}

func (c *Context) newMemoizeSpec(digits func() int) numberSpec {
	if c == nil {
		return newMemoizeSpec(digits, globalNumSpecContext)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctxt.Closed() {
		panic("Context closed")
	}
	result := newMemoizeSpec(digits, &c.ctxt)
	c.specs = append(c.specs, result)
	return result
}

// Number is a reference to a non-negative real number.
// A non-zero Number is of the form mantissa * 10^exponent
// where mantissa is between 0.1 inclusive and 1.0 exclusive. A Number
// can represent either a finite or infinite number of digits. A Number
// computes the digits of its mantissa lazily on an as needed basis. To
// compute a given digit, a Number must compute all digits that come before
// that digit. A Number stores its computed digits so that they only have
// to be computed once. Number instances are safe to use with multiple
// goroutines.
//
// The Number factory functions such as the Sqrt and CubeRoot functions
// return new Number instances that contain no computed digits initially.
//
// Because Number instances store their computed digits, it is best to
// reuse a Number instance when possible. For example the code:
//
//	n := sqrt.Sqrt(6)
//	fmt.Println(n.At(10000))
//	fmt.Println(n.At(10001))
//
// runs faster than the code:
//
//	fmt.Println(sqrt.Sqrt(6).At(10000))
//	fmt.Println(sqrt.Sqrt(6).At(10001))
//
// In the first code block, the second line reuses digits computed in the
// first line, but in the second code block, no reuse is possible since
// sqrt.Sqrt(6) always returns a Number with no precomputed digits.
//
// A Number can be 0, in which case IsZero() returns true. Zero Numbers
// have an exponent of 0 and no digits in their mantissa. This means that
// calling At() on a zero Number always returns -1. Likewise calling
// All() or Values() on a zero Number returns an empty iterator.
// However, calling String() on a zero Number returns "0" and
// printing a zero Number prints 0 according to the format specification
// used.
type Number interface {
	Sequence

	// At returns the significant digit of this Number at the given 0 based
	// position. If this Number has posit or fewer significant digits, At
	// returns -1. If posit is negative, At returns -1.
	At(posit int) int

	// WithSignificant returns a view of this Number that has no more than
	// limit significant digits. WithSignificant rounds the returned value
	// down toward zero. WithSignificant panics if limit is negative.
	WithSignificant(limit int) *FiniteNumber

	// Exponent returns the exponent of this Number.
	Exponent() int

	// Format prints this Number with the f, F, g, G, e, E verbs. The
	// verbs work in the usual way except that they always round down.
	// Because Number can have an infinite number of digits, g with no
	// precision shows a max of 16 significant digits. Format supports
	// width, precision, and the '-' flag for left justification. The v
	// verb is an alias for g.
	Format(state fmt.State, verb rune)

	// String returns the decimal representation of this Number using %g.
	String() string

	// IsZero returns true if this Number is zero.
	IsZero() bool

	withExponent(e int) Number
}

// Sqrt returns the square root of radican. Sqrt panics if radican is
// negative.
// Prefer using Context.Sqrt for better lifecycle management of Numbers.
func Sqrt(radican int64) Number {
	return nilContext.Sqrt(radican)
}

// SqrtRat returns the square root of num / denom. denom must be positive,
// and num must be non-negative or else SqrtRat panics.
// Prefer using Context.SqrtRat for better lifecycle management of Numbers.
func SqrtRat(num, denom int64) Number {
	return nilContext.SqrtRat(num, denom)
}

// SqrtBigInt returns the square root of radican. SqrtBigInt panics if
// radican is negative.
// Prefer using Context.SqrtBigInt for better lifecycle management of Numbers.
func SqrtBigInt(radican *big.Int) Number {
	return nilContext.SqrtBigInt(radican)
}

// SqrtBigRat returns the square root of radican. The denominator of radican
// must be positive, and the numerator must be non-negative or else SqrtBigRat
// panics.
// Prefer using Context.SqrtBigRat for better lifecycle management of Numbers.
func SqrtBigRat(radican *big.Rat) Number {
	return nilContext.SqrtBigRat(radican)
}

// CubeRoot returns the cube root of radican. CubeRoot panics if radican is
// negative as Number can only hold positive results.
// Prefer using Context.CubeRoot for better lifecycle management of Numbers.
func CubeRoot(radican int64) Number {
	return nilContext.CubeRoot(radican)
}

// CubeRootRat returns the cube root of num / denom. Because Number can only
// hold positive results, denom must be positive, and num must be non-negative
// or else CubeRootRat panics.
// Prefer using Context.CubeRootRat for better lifecycle management of Numbers.
func CubeRootRat(num, denom int64) Number {
	return nilContext.CubeRootRat(num, denom)
}

// CubeRootBigInt returns the cube root of radican. CubeRootBigInt panics if
// radican is negative as Number can only hold positive results.
// Prefer using Context.CubeRootBigInt for better lifecycle management of
// Numbers.
func CubeRootBigInt(radican *big.Int) Number {
	return nilContext.CubeRootBigInt(radican)
}

// CubeRootBigRat returns the cube root of radican. Because Number can only
// hold positive results, the denominator of radican must be positive, and the
// numerator must be non-negative or else CubeRootBigRat panics.
// Prefer using Context.CubeRootBigRat for better lifecycle management of
// Numbers.
func CubeRootBigRat(radican *big.Rat) Number {
	return nilContext.CubeRootBigRat(radican)
}

// NewNumberForTesting creates an arbitrary Number for testing. fixed are
// digits between 0 and 9 representing the non repeating digits that come
// immediately after the decimal place of the mantissa. repeating are digits
// between 0 and 9 representing the repeating digits that follow the non
// repeating digits of the mantissa. exp is the exponent part of the
// returned Number. NewNumberForTesting returns an error if fixed or
// repeating contain values not between 0 and 9, or if the first digit of
// the mantissa would be zero since mantissas must be between 0.1 inclusive
// and 1.0 exclusive.
// Prefer using Context.NewNumberForTesting for better lifecycle management of
// Numbers.
func NewNumberForTesting(fixed, repeating []int, exp int) (Number, error) {
	return nilContext.NewNumberForTesting(fixed, repeating, exp)
}

// NewNumber returns a new Number based on g. Although g is expected to
// follow the contract of Generator, if g yields mantissa digits outside the
// range of 0 and 9, NewNumber regards that as a signal that there are no
// more mantissa digits. Also if g happens to yield 0 as the first digit
// of the mantissa, NewNumber will return zero.
// Prefer using Context.NewNumber for better lifecycle management of Numbers.
func NewNumber(g Generator) Number {
	return nilContext.NewNumber(g)
}

// FiniteNumber is a Number with a finite number of digits. FiniteNumber
// implements both Number and FiniteSequence. The zero value for FiniteNumber
// is 0.
//
// Pass FiniteNumber instances by reference not by value. Copying a
// FiniteNumber instance is not supported and may cause errors.
type FiniteNumber struct {
	mantissa mantissa
	exponent int
}

// NewFiniteNumber works like NewNumberForTesting except that it
// returns a *FiniteNumber instead of a Number. Note that there is no
// repeating parameter because FiniteNumbers have a finite number of
// digits.
// Prefer using Context.NewFiniteNumber for better lifecycle management of
// Numbers.
func NewFiniteNumber(fixed []int, exponent int) (*FiniteNumber, error) {
	return nilContext.NewFiniteNumber(fixed, exponent)
}

// WithStart comes from the Sequence interface.
func (n *FiniteNumber) WithStart(start int) Sequence {
	return n.FiniteWithStart(start)
}

// FiniteWithStart comes from the FiniteSequence interface.
func (n *FiniteNumber) FiniteWithStart(start int) FiniteSequence {
	if start <= 0 {
		return n
	}
	return &mantissaWithStart{
		mantissa: n.mantissa,
		start:    start,
	}
}

// WithEnd comes from the Sequence interface.
func (n *FiniteNumber) WithEnd(end int) FiniteSequence {
	return n.withMantissa(n.mantissa.WithLimit(end))
}

// At comes from the Number interface.
func (n *FiniteNumber) At(posit int) int {
	return n.mantissa.At(posit)
}

// WithSignificant comes from the Number interface.
func (n *FiniteNumber) WithSignificant(limit int) *FiniteNumber {
	if limit < 0 {
		panic("limit must be non-negative")
	}
	return n.withMantissa(n.mantissa.WithLimit(limit))
}

// Exponent comes from the Number interface.
func (n *FiniteNumber) Exponent() int {
	return n.exponent
}

// Format comes from the Number interface.
func (n *FiniteNumber) Format(state fmt.State, verb rune) {
	formatSpec, ok := newFormatSpec(state, verb, n.exponent)
	if !ok {
		fmt.Fprintf(state, "%%!%c(number=%s)", verb, n.String())
		return
	}
	formatSpec.PrintField(state, n)
}

// Exact works like String, but uses enough significant digits to return
// the exact representation of n.
func (n *FiniteNumber) Exact() string {
	var builder strings.Builder
	fs := formatSpecForG(math.MaxInt, n.exponent, false)
	fs.PrintNumber(&builder, n)
	return builder.String()
}

// String comes from the Number interface.
func (n *FiniteNumber) String() string {
	var builder strings.Builder
	fs := formatSpecForG(gPrecision, n.exponent, false)
	fs.PrintNumber(&builder, n)
	return builder.String()
}

// IsZero comes from the Number interface.
func (n *FiniteNumber) IsZero() bool {
	return n.mantissa.IsZero()
}

// All comes from the Sequence interface.
func (n *FiniteNumber) All() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		n.mantissa.Scan(0, yield)
	}
}

// AllInRange comes from the Sequence interface.
func (n *FiniteNumber) AllInRange(start, end int) iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		n.mantissa.ScanInRange(0, start, end, yield)
	}
}

// Values comes from the Sequence interface.
func (n *FiniteNumber) Values() iter.Seq[int] {
	return func(yield func(value int) bool) {
		n.mantissa.ScanValues(0, yield)
	}
}

// Backward comes from the FiniteSequence interface.
func (n *FiniteNumber) Backward() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		n.mantissa.ReverseScan(0, yield)
	}
}

func (n *FiniteNumber) withExponent(e int) Number {
	if e == n.exponent || n.IsZero() {
		return n
	}
	return &FiniteNumber{exponent: e, mantissa: n.mantissa}
}

func (n *FiniteNumber) withMantissa(newMantissa mantissa) *FiniteNumber {
	if newMantissa == n.mantissa {
		return n
	}
	if newMantissa.IsZero() {
		return zeroNumber
	}
	return &FiniteNumber{mantissa: newMantissa, exponent: n.exponent}
}

func (n *FiniteNumber) private() {
}

func checkNumDenom(num, denom *big.Int) {
	if denom.Sign() <= 0 {
		panic("Denominator must be positive")
	}
	if num.Sign() < 0 {
		panic("Numerator must be non-negative")
	}
}

type mantissa struct {
	spec numberSpec
}

func (m mantissa) At(posit int) int {
	if m.spec == nil {
		return -1
	}
	return m.spec.At(posit)
}

func (m mantissa) IsZero() bool {
	return m.spec == nil
}

func (m mantissa) ReverseScan(start int, yield func(index, value int) bool) {
	digits := m.allDigits()
	for index := len(digits) - 1; index >= start; index-- {
		if !yield(index, int(digits[index])) {
			return
		}
	}
}

func (m mantissa) Scan(index int, yield func(index, value int) bool) {
	if m.spec == nil {
		return
	}
	m.spec.Scan(index, math.MaxInt, yield)
}

func (m mantissa) ScanInRange(index, start, end int, yield func(index, value int) bool) {
	if m.spec == nil {
		return
	}
	m.spec.Scan(max(index, start), end, yield)
}

func (m mantissa) ScanValues(index int, yield func(value int) bool) {
	if m.spec == nil {
		return
	}
	m.spec.ScanValues(index, math.MaxInt, yield)
}

func (m mantissa) Values() iter.Seq[int] {
	return func(yield func(int) bool) {
		m.ScanValues(0, yield)
	}
}

func (m mantissa) WithLimit(limit int) mantissa {
	return mantissa{spec: withLimit(m.spec, limit)}
}

func (m mantissa) allDigits() []int8 {
	if m.spec == nil {
		return nil
	}
	return m.spec.FirstN(math.MaxInt)
}

type formatSpec struct {
	sigDigits       int
	exactDigitCount bool
	sci             bool
	capital         bool
}

func newFormatSpec(state fmt.State, verb rune, exponent int) (
	formatSpec, bool) {
	precision, precisionOk := state.Precision()
	switch verb {
	case 'f', 'F':
		if !precisionOk {
			precision = fPrecision
		}
		return formatSpecForF(precision, exponent), true
	case 'g', 'G', 'v':
		if !precisionOk {
			precision = gPrecision
		}
		return formatSpecForG(precision, exponent, verb == 'G'), true
	case 'e', 'E':
		if !precisionOk {
			precision = fPrecision
		}
		return formatSpecForE(precision, verb == 'E'), true
	default:
		return formatSpec{}, false
	}
}

func formatSpecForF(precision, exponent int) formatSpec {
	sigDigits := precision + exponent
	return formatSpec{sigDigits: sigDigits, exactDigitCount: true}
}

func formatSpecForG(precision, exponent int, capital bool) formatSpec {
	sigDigits := precision
	if sigDigits == 0 {
		sigDigits = 1
	}
	sci := sigDigits < exponent || bigExponent(exponent)
	return formatSpec{sigDigits: sigDigits, sci: sci, capital: capital}
}

func formatSpecForE(precision int, capital bool) formatSpec {
	return formatSpec{
		sigDigits:       precision,
		exactDigitCount: true,
		sci:             true,
		capital:         capital}
}

func (f formatSpec) PrintField(state fmt.State, n *FiniteNumber) {
	width, widthOk := state.Width()
	if !widthOk {
		f.PrintNumber(state, n)
		return
	}
	var builder strings.Builder
	f.PrintNumber(&builder, n)
	field := builder.String()
	if !state.Flag('-') && len(field) < width {
		fmt.Fprint(state, strings.Repeat(" ", width-len(field)))
	}
	fmt.Fprint(state, field)
	if state.Flag('-') && len(field) < width {
		fmt.Fprint(state, strings.Repeat(" ", width-len(field)))
	}
}

func (f formatSpec) PrintNumber(w io.Writer, n *FiniteNumber) {
	if f.sci {
		sep := "e"
		if f.capital {
			sep = "E"
		}
		f.printSci(w, n.mantissa, n.exponent, sep)
	} else {
		f.printFixed(w, n.mantissa, n.exponent)
	}
}

func (f formatSpec) printFixed(w io.Writer, m mantissa, exponent int) {
	formatter := newFormatter(w, f.sigDigits, exponent, f.exactDigitCount)
	fromMantissa(m, formatter)
	formatter.Finish()
}

func (f formatSpec) printSci(
	w io.Writer, m mantissa, exponent int, sep string) {
	f.printFixed(w, m, 0)
	fmt.Fprint(w, sep)
	fmt.Fprintf(w, "%+03d", exponent)
}

func fromMantissa(m mantissa, formatter *formatter) {
	if !formatter.CanConsume() {
		return
	}
	for digit := range m.Values() {
		formatter.Consume(digit)
		if !formatter.CanConsume() {
			return
		}
	}
}

func bigExponent(exponent int) bool {
	return exponent < -3 || exponent > 6
}

type mantissaWithStart struct {
	mantissa mantissa
	start    int
}

func (m *mantissaWithStart) All() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		m.mantissa.Scan(m.start, yield)
	}
}

func (m *mantissaWithStart) AllInRange(start, end int) iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		m.mantissa.ScanInRange(m.start, start, end, yield)
	}
}

func (m *mantissaWithStart) Values() iter.Seq[int] {
	return func(yield func(value int) bool) {
		m.mantissa.ScanValues(m.start, yield)
	}
}

func (m *mantissaWithStart) Backward() iter.Seq2[int, int] {
	return func(yield func(index, value int) bool) {
		m.mantissa.ReverseScan(m.start, yield)
	}
}

func (m *mantissaWithStart) WithStart(start int) Sequence {
	return m.FiniteWithStart(start)
}

func (m *mantissaWithStart) FiniteWithStart(start int) FiniteSequence {
	if start <= m.start {
		return m
	}
	return &mantissaWithStart{mantissa: m.mantissa, start: start}
}

func (m *mantissaWithStart) WithEnd(end int) FiniteSequence {
	return m.withMantissa(m.mantissa.WithLimit(end))
}

func (m *mantissaWithStart) withMantissa(mantissa mantissa) *mantissaWithStart {
	if mantissa == m.mantissa {
		return m
	}
	return &mantissaWithStart{mantissa: mantissa, start: m.start}
}

func (m *mantissaWithStart) private() {
}

func opaqueNumber(n Number) Number {
	if _, ok := n.(*opqNumber); ok {
		return n
	}
	return &opqNumber{Number: n}
}

type opqNumber struct {
	Number
}

func (n *opqNumber) WithStart(start int) Sequence {
	result := n.Number.WithStart(start)
	if result == n.Number {
		return n
	}
	return opaqueSequence(result)
}

func (n *opqNumber) withExponent(e int) Number {
	result := n.Number.withExponent(e)
	if result == n.Number {
		return n
	}
	return opaqueNumber(result)
}

func firstAndThen(first int, next func() int) func() int {
	firstTime := true
	return func() int {
		if firstTime {
			firstTime = false
			return first
		}
		return next()
	}
}

func validDigits(x []int) bool {
	for _, d := range x {
		if digitOutOfRange(d) {
			return false
		}
	}
	return true
}
