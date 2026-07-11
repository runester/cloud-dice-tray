package dice

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func deterministic(values ...byte) *bytes.Reader {
	data := make([]byte, len(values)*8)
	for i, value := range values {
		data[i*8] = value
	}
	return bytes.NewReader(data)
}

func evaluateWith(t *testing.T, source string, rolls ...byte) Result {
	t.Helper()
	expression, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse(%q): %v", source, err)
	}
	result, err := expression.EvaluateWithReader(deterministic(rolls...))
	if err != nil {
		t.Fatalf("Evaluate(%q): %v", source, err)
	}
	return result
}

func TestArithmeticPrecedenceAndDivision(t *testing.T) {
	result := evaluateWith(t, "1 + 6 / 4 * 2")
	if got := result.Value.String(); got != "4" {
		t.Fatalf("got %s, want 4", got)
	}

	result = evaluateWith(t, "8 / 4 / 2")
	if got := result.Value.String(); got != "1" {
		t.Fatalf("got %s, want 1", got)
	}

	result = evaluateWith(t, "5 / 2")
	if got := result.Value.String(); got != "2.5" {
		t.Fatalf("got %s, want 2.5", got)
	}
	value, ok := result.Value.Scalar()
	if !ok || value.IsInt() {
		t.Fatal("division should produce a float even when its displayed result is integral")
	}
}

func TestDiceDefaultsCaseAndTrace(t *testing.T) {
	result := evaluateWith(t, "d + 1d + sum(2DF) + D20", 0, 5, 0, 5, 19)
	if got := result.Value.String(); got != "27" {
		t.Fatalf("got %s, want 27", got)
	}
	if len(result.Rolls) != 4 {
		t.Fatalf("got %d roll groups, want 4", len(result.Rolls))
	}
	if result.Rolls[2].Notation != "2dF" || numbersString(result.Rolls[2].Values) != "[-1, 1]" {
		t.Fatalf("unexpected Fudge trace: %+v", result.Rolls[2])
	}
	if result.Expression != "d + 1d + sum(2DF) + D20" {
		t.Fatalf("expression was not retained")
	}
}

func TestListArithmeticRules(t *testing.T) {
	result := evaluateWith(t, "d20 + 5", 9)
	if got := result.Value.String(); got != "15" {
		t.Fatalf("got %s, want 15", got)
	}

	// This deterministic type error must be rejected during validation, before
	// any roll is attempted.
	_, err := Parse("2d6 + 5")
	assertCode(t, err, "list_in_arithmetic")

	_, err = Parse("round(2d6)")
	assertCode(t, err, "scalar_required")
}

func TestListsFlattenAndFunctions(t *testing.T) {
	tests := map[string]string{
		"1, (2, 3), 4":           "[1, 2, 3, 4]",
		"SuM(1, 2, 3.5)":         "6.5",
		"count(1, (2, 3))":       "3",
		"min(-5, -2)":            "-5",
		"max(-5, -2)":            "-2",
		"maxk(2, 1, 4, 3)":       "[4, 3]",
		"mink(9, 3, 1, 2)":       "[1, 2, 3]",
		"equals(2, 1, 2, 2)":     "[2, 2]",
		"above(5, 1, 2)":         "[]",
		"below(2, 1, 2, 3)":      "[1]",
		"sum(above(5, 1, 2))":    "0",
		"count(equals(9, 1, 2))": "0",
	}
	for source, want := range tests {
		t.Run(source, func(t *testing.T) {
			result := evaluateWith(t, source)
			if got := result.Value.String(); got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}

func TestRounding(t *testing.T) {
	tests := map[string]string{
		"round(2.5)": "3", "round(-2.5)": "-3",
		"floor(-2.1)": "-3", "ceil(-2.1)": "-2",
	}
	for source, want := range tests {
		result := evaluateWith(t, source)
		if got := result.Value.String(); got != want {
			t.Errorf("%s = %s, want %s", source, got, want)
		}
	}
}

func TestValidationDoesNotRoll(t *testing.T) {
	expression, err := Parse("2d6")
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate("2d6"); err != nil {
		t.Fatal(err)
	}
	first, err := expression.EvaluateWithReader(deterministic(0, 1))
	if err != nil {
		t.Fatal(err)
	}
	second, err := expression.EvaluateWithReader(deterministic(5, 4))
	if err != nil {
		t.Fatal(err)
	}
	if first.Value.String() != "[1, 2]" || second.Value.String() != "[6, 5]" {
		t.Fatalf("unexpected rolls: %s, %s", first.Value.String(), second.Value.String())
	}
}

func TestLimitsAndSyntaxErrors(t *testing.T) {
	tests := []struct{ source, code string }{
		{"", "empty_expression"},
		{strings.Repeat("1", 257), "expression_too_long"},
		{"0d6", "dice_count_out_of_range"},
		{"26d6", "dice_count_out_of_range"},
		{"d1", "dice_sides_out_of_range"},
		{"d121", "dice_sides_out_of_range"},
		{"20d6, 6d6", "too_many_dice"},
		{"1 +", "missing_operand"},
		{"sum(1", "missing_closing_parenthesis"},
		{"wat(1)", "unknown_function"},
		{"2d(6)", "unexpected_token"},
		{"maxk(0, 1, 2)", "invalid_k"},
		{"maxk(1.5, 1, 2)", "invalid_k"},
		{"1 / 0", "division_by_zero"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			if test.code == "division_by_zero" {
				_, err := Evaluate(test.source)
				assertCode(t, err, test.code)
			} else {
				_, err := Parse(test.source)
				assertCode(t, err, test.code)
			}
		})
	}
}

func TestNestingLimit(t *testing.T) {
	valid := strings.Repeat("(", 10) + "1" + strings.Repeat(")", 10)
	if _, err := Parse(valid); err != nil {
		t.Fatalf("ten levels should be valid: %v", err)
	}
	invalid := "(" + valid + ")"
	_, err := Parse(invalid)
	assertCode(t, err, "nesting_too_deep")
}

func TestRandomSourceFailure(t *testing.T) {
	expression, err := Parse("d6")
	if err != nil {
		t.Fatal(err)
	}
	_, err = expression.EvaluateWithReader(bytes.NewReader(nil))
	assertCode(t, err, "random_source_error")
}

func numbersString(numbers []Number) string { return List(numbers...).String() }

func assertCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("wanted error code %q, got nil", want)
	}
	var expressionErr *Error
	if !errors.As(err, &expressionErr) {
		t.Fatalf("got error type %T, want *dice.Error", err)
	}
	if expressionErr.Code != want {
		t.Fatalf("got error code %q (%v), want %q", expressionErr.Code, err, want)
	}
}
