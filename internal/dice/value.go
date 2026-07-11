package dice

import (
	"math"
	"strconv"
	"strings"
)

// Number is a numeric DSL value. Integers remain distinguishable from floats.
type Number struct {
	integer bool
	i       int64
	f       float64
}

func Int(value int64) Number          { return Number{integer: true, i: value, f: float64(value)} }
func Float(value float64) Number      { return Number{f: value} }
func (n Number) IsInt() bool          { return n.integer }
func (n Number) Int64() (int64, bool) { return n.i, n.integer }
func (n Number) Float64() float64     { return n.f }

func (n Number) String() string {
	if n.integer {
		return strconv.FormatInt(n.i, 10)
	}
	if n.f == 0 { // avoid displaying negative zero
		return "0"
	}
	return strconv.FormatFloat(n.f, 'g', -1, 64)
}

func (n Number) equal(other Number) bool { return n.f == other.f }

// Value is either a scalar number or a (possibly empty) flat list of numbers.
type Value struct {
	list    bool
	numbers []Number
}

func Scalar(number Number) Value { return Value{numbers: []Number{number}} }

func List(numbers ...Number) Value {
	copyOfNumbers := append([]Number(nil), numbers...)
	return Value{list: true, numbers: copyOfNumbers}
}

func (v Value) IsList() bool { return v.list }

func (v Value) Numbers() []Number { return append([]Number(nil), v.numbers...) }

func (v Value) Scalar() (Number, bool) {
	if v.list || len(v.numbers) != 1 {
		return Number{}, false
	}
	return v.numbers[0], true
}

func (v Value) String() string {
	if !v.list {
		if len(v.numbers) == 0 {
			return ""
		}
		return v.numbers[0].String()
	}
	parts := make([]string, len(v.numbers))
	for i, number := range v.numbers {
		parts[i] = number.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func numberFromFloat(value float64) Number {
	if value >= math.MinInt64 && value <= math.MaxInt64 && math.Trunc(value) == value {
		return Int(int64(value))
	}
	return Float(value)
}
