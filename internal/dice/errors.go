package dice

import "fmt"

// Error is a structured expression error suitable for display to a user.
type Error struct {
	Code    string
	Message string
	Start   int // zero-based byte offset, inclusive
	End     int // zero-based byte offset, exclusive
}

func (e *Error) Error() string {
	if e.Start >= 0 {
		return fmt.Sprintf("%s: %s (at characters %d–%d)", e.Code, e.Message, e.Start+1, e.End)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func expressionError(code, message string, start, end int) *Error {
	return &Error{Code: code, Message: message, Start: start, End: end}
}
