package dice

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxExpressionLength = 256
	MaxDice             = 25
	MaxSides            = 120
	MaxNesting          = 10
)

type node interface {
	span() (int, int)
}

type numberNode struct {
	value      Number
	start, end int
}

func (n *numberNode) span() (int, int) { return n.start, n.end }

type diceNode struct {
	count, sides int
	fudge        bool
	notation     string
	start, end   int
}

func (n *diceNode) span() (int, int) { return n.start, n.end }

type unaryNode struct {
	operator   tokenKind
	operand    node
	start, end int
}

func (n *unaryNode) span() (int, int) { return n.start, n.end }

type binaryNode struct {
	operator    tokenKind
	left, right node
	start, end  int
}

func (n *binaryNode) span() (int, int) { return n.start, n.end }

type listNode struct {
	elements   []node
	start, end int
}

func (n *listNode) span() (int, int) { return n.start, n.end }

type callNode struct {
	name       string
	arguments  []node
	start, end int
}

func (n *callNode) span() (int, int) { return n.start, n.end }

// Expression is a parsed and validated expression. It is safe to evaluate
// repeatedly; each evaluation performs fresh rolls.
type Expression struct {
	source string
	root   node
}

func (e *Expression) Source() string { return e.source }

// Parse validates syntax and all statically knowable limits without rolling.
func Parse(source string) (*Expression, error) {
	if utf8.RuneCountInString(source) > MaxExpressionLength {
		return nil, expressionError("expression_too_long", fmt.Sprintf("expression must not exceed %d characters", MaxExpressionLength), 0, len(source))
	}
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := parser{tokens: tokens}
	root, err := p.parseList(0)
	if err != nil {
		return nil, err
	}
	if p.current().kind != tokenEOF {
		t := p.current()
		return nil, expressionError("unexpected_token", fmt.Sprintf("unexpected %q", t.text), t.start, t.end)
	}
	if root == nil {
		return nil, expressionError("empty_expression", "expression cannot be empty", 0, 0)
	}
	if p.dice > MaxDice {
		return nil, expressionError("too_many_dice", fmt.Sprintf("expression may roll at most %d dice", MaxDice), 0, len(source))
	}
	if err := validateSemantics(root); err != nil {
		return nil, err
	}
	return &Expression{source: source, root: root}, nil
}

// Validate checks an expression without consuming randomness.
func Validate(source string) error { _, err := Parse(source); return err }

type parser struct {
	tokens         []token
	position, dice int
}

func (p *parser) current() token { return p.tokens[p.position] }
func (p *parser) advance() token {
	t := p.current()
	if t.kind != tokenEOF {
		p.position++
	}
	return t
}

func (p *parser) parseList(depth int) (node, error) {
	first, err := p.parseAdditive(depth)
	if err != nil || first == nil {
		return first, err
	}
	elements := []node{first}
	for p.current().kind == tokenComma {
		comma := p.advance()
		if p.current().kind == tokenEOF || p.current().kind == tokenRightParen || p.current().kind == tokenComma {
			return nil, expressionError("missing_operand", "expected an expression after comma", comma.start, comma.end)
		}
		next, err := p.parseAdditive(depth)
		if err != nil {
			return nil, err
		}
		elements = append(elements, next)
	}
	if len(elements) == 1 {
		return first, nil
	}
	start, _ := first.span()
	_, end := elements[len(elements)-1].span()
	return &listNode{elements: elements, start: start, end: end}, nil
}

func (p *parser) parseAdditive(depth int) (node, error) {
	left, err := p.parseMultiplicative(depth)
	if err != nil {
		return nil, err
	}
	for p.current().kind == tokenPlus || p.current().kind == tokenMinus {
		op := p.advance()
		right, err := p.parseMultiplicative(depth)
		if err != nil {
			return nil, err
		}
		if right == nil {
			return nil, expressionError("missing_operand", "expected an expression after operator", op.start, op.end)
		}
		start, _ := left.span()
		_, end := right.span()
		left = &binaryNode{operator: op.kind, left: left, right: right, start: start, end: end}
	}
	return left, nil
}

func (p *parser) parseMultiplicative(depth int) (node, error) {
	left, err := p.parseUnary(depth)
	if err != nil {
		return nil, err
	}
	for p.current().kind == tokenStar || p.current().kind == tokenSlash {
		op := p.advance()
		right, err := p.parseUnary(depth)
		if err != nil {
			return nil, err
		}
		if right == nil {
			return nil, expressionError("missing_operand", "expected an expression after operator", op.start, op.end)
		}
		start, _ := left.span()
		_, end := right.span()
		left = &binaryNode{operator: op.kind, left: left, right: right, start: start, end: end}
	}
	return left, nil
}

func (p *parser) parseUnary(depth int) (node, error) {
	if p.current().kind == tokenPlus || p.current().kind == tokenMinus {
		op := p.advance()
		operand, err := p.parseUnary(depth)
		if err != nil {
			return nil, err
		}
		if operand == nil {
			return nil, expressionError("missing_operand", "expected an expression after unary operator", op.start, op.end)
		}
		_, end := operand.span()
		return &unaryNode{operator: op.kind, operand: operand, start: op.start, end: end}, nil
	}
	return p.parseDice(depth)
}

func (p *parser) parseDice(depth int) (node, error) {
	t := p.current()
	count := 1
	start := t.start
	if t.kind == tokenNumber && p.position+1 < len(p.tokens) && (p.tokens[p.position+1].kind == tokenDice || p.tokens[p.position+1].kind == tokenFudgeDice) {
		if !t.number.IsInt() {
			return nil, expressionError("invalid_dice_count", "dice count must be an integer", t.start, t.end)
		}
		value, _ := t.number.Int64()
		count = int(value)
		p.advance()
		t = p.current()
	}
	if t.kind != tokenDice && t.kind != tokenFudgeDice {
		return p.parsePrimary(depth)
	}
	p.advance()
	if count < 1 || count > MaxDice {
		return nil, expressionError("dice_count_out_of_range", fmt.Sprintf("dice count must be between 1 and %d", MaxDice), start, t.end)
	}
	fudge := t.kind == tokenFudgeDice
	sides := 6
	end := t.end
	if !fudge && p.current().kind == tokenNumber {
		sideToken := p.advance()
		end = sideToken.end
		if !sideToken.number.IsInt() {
			return nil, expressionError("invalid_dice_sides", "number of sides must be an integer", sideToken.start, sideToken.end)
		}
		value, _ := sideToken.number.Int64()
		sides = int(value)
	}
	if sides < 2 || sides > MaxSides {
		return nil, expressionError("dice_sides_out_of_range", fmt.Sprintf("dice sides must be between 2 and %d", MaxSides), start, end)
	}
	p.dice += count
	notation := fmt.Sprintf("%dd%d", count, sides)
	if fudge {
		notation = fmt.Sprintf("%ddF", count)
	}
	return &diceNode{count: count, sides: sides, fudge: fudge, notation: notation, start: start, end: end}, nil
}

func (p *parser) parsePrimary(depth int) (node, error) {
	t := p.current()
	switch t.kind {
	case tokenNumber:
		p.advance()
		return &numberNode{value: t.number, start: t.start, end: t.end}, nil
	case tokenIdentifier:
		p.advance()
		if p.current().kind != tokenLeftParen {
			return nil, expressionError("unknown_identifier", fmt.Sprintf("unknown identifier %q", t.text), t.start, t.end)
		}
		return p.parseCall(t, depth)
	case tokenLeftParen:
		if depth >= MaxNesting {
			return nil, expressionError("nesting_too_deep", fmt.Sprintf("expressions may be nested at most %d levels", MaxNesting), t.start, t.end)
		}
		p.advance()
		expression, err := p.parseList(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.current().kind != tokenRightParen {
			return nil, expressionError("missing_closing_parenthesis", "expected closing parenthesis", t.start, t.end)
		}
		p.advance()
		return expression, nil
	case tokenEOF, tokenRightParen, tokenComma:
		return nil, nil
	default:
		return nil, expressionError("unexpected_token", fmt.Sprintf("unexpected %q", t.text), t.start, t.end)
	}
}

func (p *parser) parseCall(name token, depth int) (node, error) {
	if depth >= MaxNesting {
		return nil, expressionError("nesting_too_deep", fmt.Sprintf("expressions may be nested at most %d levels", MaxNesting), name.start, name.end)
	}
	p.advance() // (
	var args []node
	if p.current().kind != tokenRightParen {
		argument, err := p.parseList(depth + 1)
		if err != nil {
			return nil, err
		}
		if argument == nil {
			return nil, expressionError("missing_argument", "function requires an argument", name.start, name.end)
		}
		if list, ok := argument.(*listNode); ok {
			args = list.elements
		} else {
			args = []node{argument}
		}
	}
	if p.current().kind != tokenRightParen {
		return nil, expressionError("missing_closing_parenthesis", "expected closing parenthesis", name.start, name.end)
	}
	end := p.advance().end
	function := strings.ToLower(name.text)
	if _, ok := functionNames[function]; !ok {
		return nil, expressionError("unknown_function", fmt.Sprintf("unknown function %q", name.text), name.start, name.end)
	}
	if err := validateArity(function, len(args), name); err != nil {
		return nil, err
	}
	return &callNode{name: function, arguments: args, start: name.start, end: end}, nil
}

var functionNames = map[string]struct{}{"sum": {}, "min": {}, "max": {}, "count": {}, "maxk": {}, "mink": {}, "equals": {}, "above": {}, "below": {}, "round": {}, "floor": {}, "ceil": {}}

func validateArity(name string, count int, at token) error {
	switch name {
	case "round", "floor", "ceil":
		if count != 1 {
			return expressionError("wrong_argument_count", fmt.Sprintf("%s expects exactly one argument", name), at.start, at.end)
		}
	case "maxk", "mink", "equals", "above", "below":
		if count < 2 {
			return expressionError("wrong_argument_count", fmt.Sprintf("%s expects a parameter followed by one or more values", name), at.start, at.end)
		}
	}
	return nil
}
