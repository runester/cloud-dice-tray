package dice

import "fmt"

// valueShape describes the number of values a node can produce. Arithmetic
// accepts scalars and lists only when they contain exactly one value.
type valueShape struct {
	min, max int
}

func scalarShape() valueShape { return valueShape{min: 1, max: 1} }

func validateSemantics(root node) error {
	_, err := inferShape(root)
	return err
}

func inferShape(n node) (valueShape, error) {
	switch current := n.(type) {
	case *numberNode:
		return scalarShape(), nil
	case *diceNode:
		return valueShape{min: current.count, max: current.count}, nil
	case *listNode:
		shape := valueShape{}
		for _, element := range current.elements {
			elementShape, err := inferShape(element)
			if err != nil {
				return valueShape{}, err
			}
			shape.min += elementShape.min
			shape.max += elementShape.max
		}
		return shape, nil
	case *unaryNode:
		shape, err := inferShape(current.operand)
		if err != nil {
			return valueShape{}, err
		}
		if !shape.isExactlyOne() {
			return valueShape{}, scalarShapeError(current.start, current.end)
		}
		return scalarShape(), nil
	case *binaryNode:
		left, err := inferShape(current.left)
		if err != nil {
			return valueShape{}, err
		}
		right, err := inferShape(current.right)
		if err != nil {
			return valueShape{}, err
		}
		if !left.isExactlyOne() || !right.isExactlyOne() {
			return valueShape{}, scalarShapeError(current.start, current.end)
		}
		return scalarShape(), nil
	case *callNode:
		return inferCallShape(current)
	default:
		return valueShape{}, expressionError("internal_error", "unknown expression node", -1, -1)
	}
}

func inferCallShape(call *callNode) (valueShape, error) {
	argumentShapes := make([]valueShape, len(call.arguments))
	for i, argument := range call.arguments {
		shape, err := inferShape(argument)
		if err != nil {
			return valueShape{}, err
		}
		argumentShapes[i] = shape
	}

	switch call.name {
	case "sum", "count":
		return scalarShape(), nil
	case "min", "max":
		combined := combineShapes(argumentShapes)
		if combined.max == 0 {
			return valueShape{}, expressionError("empty_list", call.name+" requires at least one value", call.start, call.end)
		}
		return scalarShape(), nil
	case "round", "floor", "ceil":
		if len(argumentShapes) != 1 || !argumentShapes[0].isExactlyOne() {
			return valueShape{}, expressionError("scalar_required", call.name+" requires exactly one scalar value", call.start, call.end)
		}
		return scalarShape(), nil
	case "maxk", "mink":
		if !argumentShapes[0].isExactlyOne() {
			return valueShape{}, expressionError("scalar_required", "the first argument to "+call.name+" must be scalar", call.start, call.end)
		}
		rest := combineShapes(argumentShapes[1:])
		if literal, ok := call.arguments[0].(*numberNode); ok {
			k, integer := literal.value.Int64()
			if !integer || k < 1 {
				return valueShape{}, expressionError("invalid_k", "k must be an integer greater than or equal to 1", call.start, call.end)
			}
			if int64(rest.min) >= k {
				return valueShape{min: int(k), max: int(k)}, nil
			}
			maximum := rest.max
			if int64(maximum) > k {
				maximum = int(k)
			}
			return valueShape{min: rest.min, max: maximum}, nil
		}
		return valueShape{min: 0, max: rest.max}, nil
	case "equals", "above", "below":
		if !argumentShapes[0].isExactlyOne() {
			return valueShape{}, expressionError("scalar_required", "the first argument to "+call.name+" must be scalar", call.start, call.end)
		}
		rest := combineShapes(argumentShapes[1:])
		return valueShape{min: 0, max: rest.max}, nil
	default:
		return valueShape{}, expressionError("internal_error", fmt.Sprintf("unknown function %q", call.name), call.start, call.end)
	}
}

func (s valueShape) isExactlyOne() bool { return s.min == 1 && s.max == 1 }

func combineShapes(shapes []valueShape) valueShape {
	combined := valueShape{}
	for _, shape := range shapes {
		combined.min += shape.min
		combined.max += shape.max
	}
	return combined
}

func scalarShapeError(start, end int) error {
	return expressionError("list_in_arithmetic", "arithmetic requires scalar values or a one-element list; aggregate multiple dice with a function such as sum(), min(), or max()", start, end)
}
