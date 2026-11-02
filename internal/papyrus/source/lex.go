// Package source reads Papyrus source (.psc) into the shared intermediate
// form. It's a tokenizer and a recursive-descent parser for declarations,
// statements, and expressions, with no type checking, which recovers from
// what it can't read at the next line, so one odd line costs that line.
package source

import (
	"strings"
	"unicode"
)

type kind int

const (
	tEOF kind = iota
	tNewline
	tIdent
	tString
	tInt
	tFloat
	tOp // punctuation and operators
)

type token struct {
	kind kind
	text string // an identifier as written, a string's value, or the operator
	line int
}

// lex splits source into tokens. Comments (; to the end of the line, ;/ …
// /; across lines, and { … } documentation) are dropped, a \ at the end of
// a line joins it to the next, and runs of newlines collapse to one.
func lex(src string) []token {
	src = strings.TrimPrefix(src, "\ufeff")
	var out []token
	line := 1
	emit := func(k kind, text string) {
		if k == tNewline && (len(out) == 0 || out[len(out)-1].kind == tNewline) {
			return
		}
		out = append(out, token{kind: k, text: text, line: line})
	}
	r := []rune(src)
	for i := 0; i < len(r); {
		c := r[i]
		switch {
		case c == '\n':
			emit(tNewline, "")
			line++
			i++
		case c == '\r' || c == ' ' || c == '\t' || c == '\f' || c == '\v':
			i++
		case c == '\\':
			// A line continuation: skip to past the end of the line.
			j := i + 1
			for j < len(r) && r[j] != '\n' {
				j++
			}
			if j < len(r) {
				line++
				j++
			}
			i = j
		case c == ';':
			if i+1 < len(r) && r[i+1] == '/' {
				j := i + 2
				for j+1 < len(r) && !(r[j] == '/' && r[j+1] == ';') {
					if r[j] == '\n' {
						line++
					}
					j++
				}
				i = min(j+2, len(r))
			} else {
				for i < len(r) && r[i] != '\n' {
					i++
				}
			}
		case c == '{':
			for i < len(r) && r[i] != '}' {
				if r[i] == '\n' {
					line++
				}
				i++
			}
			i++
		case c == '"':
			var sb strings.Builder
			j := i + 1
			for j < len(r) && r[j] != '"' && r[j] != '\n' {
				if r[j] == '\\' && j+1 < len(r) {
					j++
					switch r[j] {
					case 'n':
						sb.WriteRune('\n')
					case 't':
						sb.WriteRune('\t')
					default:
						sb.WriteRune(r[j])
					}
				} else {
					sb.WriteRune(r[j])
				}
				j++
			}
			emit(tString, sb.String())
			i = j + 1
		case unicode.IsDigit(c) || (c == '.' && i+1 < len(r) && unicode.IsDigit(r[i+1])):
			j := i
			if c == '0' && i+1 < len(r) && (r[i+1] == 'x' || r[i+1] == 'X') {
				j += 2
				for j < len(r) && isHex(r[j]) {
					j++
				}
				emit(tInt, string(r[i:j]))
			} else {
				float := false
				for j < len(r) && (unicode.IsDigit(r[j]) || r[j] == '.') {
					if r[j] == '.' {
						float = true
					}
					j++
				}
				if float {
					emit(tFloat, string(r[i:j]))
				} else {
					emit(tInt, string(r[i:j]))
				}
			}
			i = j
		case unicode.IsLetter(c) || c == '_' || c == ':':
			j := i
			for j < len(r) && (unicode.IsLetter(r[j]) || unicode.IsDigit(r[j]) || r[j] == '_' || r[j] == ':') {
				j++
			}
			emit(tIdent, string(r[i:j]))
			i = j
		default:
			op := string(c)
			if i+1 < len(r) {
				two := string(r[i : i+2])
				switch two {
				case "==", "!=", "<=", ">=", "&&", "||", "+=", "-=", "*=", "/=", "%=":
					op = two
				}
			}
			emit(tOp, op)
			i += len([]rune(op))
		}
	}
	emit(tNewline, "")
	out = append(out, token{kind: tEOF, line: line})
	return out
}

func isHex(c rune) bool {
	return unicode.IsDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
