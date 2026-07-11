// Package dice parses, validates, and evaluates cloud-dice-tray expressions.
//
// Parsing and validation never consume randomness. Call Parse to prepare an
// expression, then Expression.Evaluate to roll it exactly once. Evaluate is a
// convenience for both operations. Evaluation uses crypto/rand by default.
package dice
