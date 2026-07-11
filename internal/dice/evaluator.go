package dice

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
)

// Roll records one dice term and all raw results generated for it.
type Roll struct {
	Notation string
	Values   []Number
}

// Result contains the submitted expression, raw rolls, and final value.
type Result struct {
	Expression string
	Rolls      []Roll
	Value      Value
}

// Evaluate parses, validates, and evaluates source exactly once.
func Evaluate(source string) (Result, error) {
	expression, err := Parse(source)
	if err != nil {
		return Result{}, err
	}
	return expression.Evaluate()
}

// Evaluate evaluates a previously parsed expression with cryptographic randomness.
func (e *Expression) Evaluate() (Result, error) { return e.EvaluateWithReader(cryptorand.Reader) }

// EvaluateWithReader permits deterministic tests and custom secure random sources.
func (e *Expression) EvaluateWithReader(random io.Reader) (Result, error) {
	if e == nil || e.root == nil {
		return Result{}, expressionError("invalid_expression", "cannot evaluate a nil expression", -1, -1)
	}
	if random == nil {
		return Result{}, expressionError("random_source_error", "random source cannot be nil", -1, -1)
	}
	context := evaluationContext{random: random}
	value, err := context.evaluate(e.root)
	if err != nil {
		return Result{}, err
	}
	return Result{Expression: e.source, Rolls: context.rolls, Value: value}, nil
}

type evaluationContext struct {
	random io.Reader
	rolls  []Roll
}

func (c *evaluationContext) evaluate(n node) (Value, error) {
	switch current := n.(type) {
	case *numberNode:
		return Scalar(current.value), nil
	case *diceNode:
		values := make([]Number, current.count)
		for i := range values {
			face, err := unbiasedInt(c.random, current.sides)
			if err != nil {
				return Value{}, expressionError("random_source_error", "could not generate a random roll", current.start, current.end)
			}
			if current.fudge {
				face = (face / 2) - 1
			} else {
				face++
			}
			values[i] = Int(int64(face))
		}
		c.rolls = append(c.rolls, Roll{Notation: current.notation, Values: append([]Number(nil), values...)})
		return List(values...), nil
	case *listNode:
		var values []Number
		for _, element := range current.elements {
			value, err := c.evaluate(element)
			if err != nil {
				return Value{}, err
			}
			values = append(values, value.numbers...)
		}
		return List(values...), nil
	case *unaryNode:
		value, err := c.evaluate(current.operand)
		if err != nil {
			return Value{}, err
		}
		number, err := arithmeticScalar(value, current.start, current.end)
		if err != nil {
			return Value{}, err
		}
		if current.operator == tokenMinus {
			if number.integer {
				number = Int(-number.i)
			} else {
				number = Float(-number.f)
			}
		}
		return Scalar(number), nil
	case *binaryNode:
		left, err := c.evaluate(current.left)
		if err != nil {
			return Value{}, err
		}
		right, err := c.evaluate(current.right)
		if err != nil {
			return Value{}, err
		}
		return evaluateBinary(current, left, right)
	case *callNode:
		arguments := make([]Value, len(current.arguments))
		for i, argument := range current.arguments {
			value, err := c.evaluate(argument)
			if err != nil {
				return Value{}, err
			}
			arguments[i] = value
		}
		return evaluateFunction(current, arguments)
	default:
		return Value{}, expressionError("internal_error", "unknown expression node", -1, -1)
	}
}

func arithmeticScalar(value Value, start, end int) (Number, error) {
	if len(value.numbers) == 1 {
		return value.numbers[0], nil
	}
	return Number{}, expressionError("list_in_arithmetic", "arithmetic requires scalar values or a one-element list", start, end)
}

func evaluateBinary(n *binaryNode, leftValue, rightValue Value) (Value, error) {
	left, err := arithmeticScalar(leftValue, n.start, n.end)
	if err != nil {
		return Value{}, err
	}
	right, err := arithmeticScalar(rightValue, n.start, n.end)
	if err != nil {
		return Value{}, err
	}
	switch n.operator {
	case tokenPlus:
		if left.integer && right.integer {
			return Scalar(Int(left.i + right.i)), nil
		}
		return Scalar(Float(left.f + right.f)), nil
	case tokenMinus:
		if left.integer && right.integer {
			return Scalar(Int(left.i - right.i)), nil
		}
		return Scalar(Float(left.f - right.f)), nil
	case tokenStar:
		if left.integer && right.integer {
			return Scalar(Int(left.i * right.i)), nil
		}
		return Scalar(Float(left.f * right.f)), nil
	case tokenSlash:
		if right.f == 0 {
			return Value{}, expressionError("division_by_zero", "cannot divide by zero", n.start, n.end)
		}
		return Scalar(Float(left.f / right.f)), nil
	default:
		return Value{}, expressionError("internal_error", "unknown arithmetic operator", n.start, n.end)
	}
}

func flatten(values []Value) []Number {
	var result []Number
	for _, value := range values {
		result = append(result, value.numbers...)
	}
	return result
}

func evaluateFunction(call *callNode, arguments []Value) (Value, error) {
	values := flatten(arguments)
	switch call.name {
	case "sum":
		result := Int(0)
		for _, value := range values {
			if result.integer && value.integer {
				result = Int(result.i + value.i)
			} else {
				result = Float(result.f + value.f)
			}
		}
		return Scalar(result), nil
	case "count":
		return Scalar(Int(int64(len(values)))), nil
	case "min", "max":
		if len(values) == 0 {
			return Value{}, expressionError("empty_list", call.name+" requires at least one value", call.start, call.end)
		}
		result := values[0]
		for _, value := range values[1:] {
			if (call.name == "min" && value.f < result.f) || (call.name == "max" && value.f > result.f) {
				result = value
			}
		}
		return Scalar(result), nil
	case "round", "floor", "ceil":
		if len(values) != 1 {
			return Value{}, expressionError("scalar_required", call.name+" requires exactly one scalar value", call.start, call.end)
		}
		var result float64
		switch call.name {
		case "round":
			result = math.Round(values[0].f)
		case "floor":
			result = math.Floor(values[0].f)
		case "ceil":
			result = math.Ceil(values[0].f)
		}
		return Scalar(numberFromFloat(result)), nil
	case "maxk", "mink":
		parameter, rest, err := functionParameter(call, arguments)
		if err != nil {
			return Value{}, err
		}
		k, ok := parameter.Int64()
		if !ok || k < 1 {
			return Value{}, expressionError("invalid_k", "k must be an integer greater than or equal to 1", call.start, call.end)
		}
		ordered := append([]Number(nil), rest...)
		sort.SliceStable(ordered, func(i, j int) bool {
			if call.name == "maxk" {
				return ordered[i].f > ordered[j].f
			}
			return ordered[i].f < ordered[j].f
		})
		if k < int64(len(ordered)) {
			ordered = ordered[:k]
		}
		return List(ordered...), nil
	case "equals", "above", "below":
		parameter, rest, err := functionParameter(call, arguments)
		if err != nil {
			return Value{}, err
		}
		result := make([]Number, 0)
		for _, value := range rest {
			matches := call.name == "equals" && value.equal(parameter) || call.name == "above" && value.f > parameter.f || call.name == "below" && value.f < parameter.f
			if matches {
				result = append(result, value)
			}
		}
		return List(result...), nil
	}
	return Value{}, expressionError("internal_error", fmt.Sprintf("unknown function %q", call.name), call.start, call.end)
}

func functionParameter(call *callNode, arguments []Value) (Number, []Number, error) {
	if len(arguments) < 2 {
		return Number{}, nil, expressionError("wrong_argument_count", call.name+" requires a parameter and values", call.start, call.end)
	}
	parameter, err := arithmeticScalar(arguments[0], call.start, call.end)
	if err != nil {
		return Number{}, nil, expressionError("scalar_required", "the first argument to "+call.name+" must be scalar", call.start, call.end)
	}
	return parameter, flatten(arguments[1:]), nil
}

func unbiasedInt(reader io.Reader, maximum int) (int, error) {
	limit := uint64(math.MaxUint64 - (math.MaxUint64 % uint64(maximum)))
	var bytes [8]byte
	for {
		if _, err := io.ReadFull(reader, bytes[:]); err != nil {
			return 0, err
		}
		value := binary.LittleEndian.Uint64(bytes[:])
		if value < limit {
			return int(value % uint64(maximum)), nil
		}
	}
}
