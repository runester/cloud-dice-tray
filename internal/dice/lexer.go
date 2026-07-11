package dice

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenNumber
	tokenIdentifier
	tokenDice
	tokenFudgeDice
	tokenPlus
	tokenMinus
	tokenStar
	tokenSlash
	tokenComma
	tokenLeftParen
	tokenRightParen
)

type token struct {
	kind       tokenKind
	text       string
	start, end int
	number     Number
}

func allDigits(value string) bool {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
}

func lex(input string) ([]token, error) {
	var tokens []token
	for pos := 0; pos < len(input); {
		r, size := utf8.DecodeRuneInString(input[pos:])
		if unicode.IsSpace(r) {
			pos += size
			continue
		}

		start := pos
		switch r {
		case '+', '-', '*', '/', ',', '(', ')':
			kind := map[rune]tokenKind{
				'+': tokenPlus, '-': tokenMinus, '*': tokenStar, '/': tokenSlash,
				',': tokenComma, '(': tokenLeftParen, ')': tokenRightParen,
			}[r]
			pos += size
			tokens = append(tokens, token{kind: kind, text: input[start:pos], start: start, end: pos})
			continue
		}

		if unicode.IsDigit(r) || r == '.' {
			dotSeen := false
			digitSeen := false
			for pos < len(input) {
				r, size = utf8.DecodeRuneInString(input[pos:])
				if unicode.IsDigit(r) {
					digitSeen = true
					pos += size
					continue
				}
				if r == '.' && !dotSeen {
					dotSeen = true
					pos += size
					continue
				}
				break
			}
			if !digitSeen {
				return nil, expressionError("invalid_number", "a decimal point must be part of a number", start, pos)
			}
			text := input[start:pos]
			if dotSeen {
				value, err := strconv.ParseFloat(text, 64)
				if err != nil {
					return nil, expressionError("invalid_number", "invalid numeric literal", start, pos)
				}
				tokens = append(tokens, token{kind: tokenNumber, text: text, start: start, end: pos, number: Float(value)})
			} else {
				value, err := strconv.ParseInt(text, 10, 64)
				if err != nil {
					return nil, expressionError("invalid_number", "integer literal is too large", start, pos)
				}
				tokens = append(tokens, token{kind: tokenNumber, text: text, start: start, end: pos, number: Int(value)})
			}
			continue
		}

		if unicode.IsLetter(r) || r == '_' {
			pos += size
			for pos < len(input) {
				r, size = utf8.DecodeRuneInString(input[pos:])
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
					break
				}
				pos += size
			}
			text := input[start:pos]
			lower := strings.ToLower(text)
			switch {
			case lower == "d":
				tokens = append(tokens, token{kind: tokenDice, text: text, start: start, end: pos})
			case lower == "df":
				tokens = append(tokens, token{kind: tokenFudgeDice, text: text, start: start, end: pos})
			case len(lower) > 1 && lower[0] == 'd' && allDigits(lower[1:]):
				sides, err := strconv.ParseInt(lower[1:], 10, 64)
				if err != nil {
					return nil, expressionError("invalid_number", "integer literal is too large", start+1, pos)
				}
				tokens = append(tokens,
					token{kind: tokenDice, text: text[:1], start: start, end: start + 1},
					token{kind: tokenNumber, text: text[1:], start: start + 1, end: pos, number: Int(sides)},
				)
			default:
				tokens = append(tokens, token{kind: tokenIdentifier, text: text, start: start, end: pos})
			}
			continue
		}

		return nil, expressionError("invalid_character", "invalid character in expression", start, start+size)
	}
	tokens = append(tokens, token{kind: tokenEOF, start: len(input), end: len(input)})
	return tokens, nil
}
