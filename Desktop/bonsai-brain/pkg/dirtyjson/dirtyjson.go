// Package dirtyjson provides a tolerant, streaming JSON parser that can handle
// malformed or incomplete JSON commonly produced by LLMs: trailing commas,
// missing closing braces/brackets, unquoted keys, and single-quoted strings.
//
// It is implemented as a state machine (not regex) and supports incremental
// feeding of chunks followed by a final Parse() call.
package dirtyjson

import (
	"fmt"
	"strings"
	"unicode"
)

// state represents the parser's current position in the token stream.
type state int

const (
	stStart       state = iota // expecting { or [
	stKey                      // expecting a key (quoted or unquoted)
	stColon                    // expecting :
	stValue                    // expecting a value
	stString                   // inside a quoted string
	stStringEsc                // inside a string escape sequence
	stNumber                   // inside a number literal
	stBoolOrNull               // inside true/false/null
	stArray                    // inside an array
	stEnd                      // clean end
)

// DirtyJSON is a tolerant JSON parser that accumulates bytes via Feed()
// and produces a Go value via Parse().
type DirtyJSON struct {
	buf       []byte
	completed bool // set when we've seen a definitive end
}

// New creates a new DirtyJSON parser.
func New() *DirtyJSON {
	return &DirtyJSON{}
}

// Feed appends a chunk of JSON bytes to the internal buffer.
func (d *DirtyJSON) Feed(chunk []byte) {
	d.buf = append(d.buf, chunk...)
}

// FeedString is a convenience wrapper around Feed.
func (d *DirtyJSON) FeedString(s string) {
	d.buf = append(d.buf, s...)
}

// IsCompleted returns true if the parser has seen a complete top-level value
// (balanced braces/brackets or a primitive).
func (d *DirtyJSON) IsCompleted() bool {
	return d.completed || d.detectCompletion()
}

// Parse attempts to parse the accumulated buffer into a map[string]any.
// If the top-level value is an array it is returned wrapped as
// {"_array": []any{…}} so the return type is always map[string]any.
func (d *DirtyJSON) Parse() (map[string]any, error) {
	// Pre-process: repair common issues.
	repaird := d.repair(string(d.buf))
	tokens := d.tokenize(repaird)
	val, _, err := d.parseValue(tokens, 0)
	if err != nil {
		return nil, fmt.Errorf("dirtyjson: %w", err)
	}
	switch v := val.(type) {
	case map[string]any:
		return v, nil
	case []any:
		return map[string]any{"_array": v}, nil
	default:
		return map[string]any{"_value": v}, nil
	}
}

// ---------------------------------------------------------------------------
// Repair
// ---------------------------------------------------------------------------

func (d *DirtyJSON) repair(s string) string {
	// Remove trailing commas before } or ]
	s = d.removeTrailingCommas(s)
	// Quote unquoted keys
	s = d.quoteKeys(s)
	// Convert single-quoted strings to double-quoted
	s = d.fixSingleQuotes(s)
	return s
}

func (d *DirtyJSON) removeTrailingCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			// Look ahead past whitespace for } or ]
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue // skip the comma
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func (d *DirtyJSON) quoteKeys(s string) string {
	// A simple heuristic: an unquoted key is a sequence of word chars
	// followed by optional whitespace and then ':'.
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		// If we're inside a string, copy verbatim until the closing quote.
		if s[i] == '"' {
			b.WriteByte(s[i])
			i++
			for i < len(s) {
				if s[i] == '\\' {
					b.WriteByte(s[i])
					i++
					if i < len(s) {
						b.WriteByte(s[i])
						i++
					}
					continue
				}
				b.WriteByte(s[i])
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if s[i] == '\'' {
			// Single-quoted string — we'll handle in fixSingleQuotes,
			// but skip here so we don't misinterpret.
			b.WriteByte(s[i])
			i++
			for i < len(s) && s[i] != '\'' {
				if s[i] == '\\' {
					b.WriteByte(s[i])
					i++
					if i < len(s) {
						b.WriteByte(s[i])
						i++
					}
					continue
				}
				b.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				b.WriteByte(s[i]) // closing quote
				i++
			}
			continue
		}

		// Detect potential unquoted key: letter/underscore/word start
		if isUnquotedKeyStart(s[i]) {
			start := i
			for i < len(s) && isUnquotedKeyChar(s[i]) {
				i++
			}
			word := s[start:i]
			// Skip whitespace
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			if j < len(s) && s[j] == ':' {
				// It's a key — quote it.
				b.WriteByte('"')
				b.WriteString(word)
				b.WriteByte('"')
				i = j // leave colon for next iteration
			} else {
				// Not a key, emit as-is (could be a value like true/false/null).
				b.WriteString(word)
			}
			continue
		}

		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func isUnquotedKeyStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isUnquotedKeyChar(c byte) bool {
	return isUnquotedKeyStart(c) || (c >= '0' && c <= '9') || c == '-' || c == '.'
}

func (d *DirtyJSON) fixSingleQuotes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '"' {
			// Copy double-quoted string verbatim.
			b.WriteByte(s[i])
			i++
			for i < len(s) {
				if s[i] == '\\' {
					b.WriteByte(s[i])
					i++
					if i < len(s) {
						b.WriteByte(s[i])
						i++
					}
					continue
				}
				b.WriteByte(s[i])
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if s[i] == '\'' {
			// Convert single-quoted to double-quoted.
			b.WriteByte('"')
			i++
			for i < len(s) && s[i] != '\'' {
				if s[i] == '\\' {
					b.WriteByte(s[i])
					i++
					if i < len(s) {
						b.WriteByte(s[i])
						i++
					}
					continue
				}
				if s[i] == '"' {
					// Escape embedded double quotes.
					b.WriteByte('\\')
				}
				b.WriteByte(s[i])
				i++
			}
			b.WriteByte('"')
			if i < len(s) {
				i++ // skip closing quote
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Tokenizer
// ---------------------------------------------------------------------------

type tokenType int

const (
	tokLBrace tokenType = iota
	tokRBrace
	tokLBracket
	tokRBracket
	tokColon
	tokComma
	tokString
	tokNumber
	tokBoolOrNull
	tokEOF
)

type token struct {
	typ tokenType
	val string
}

func (d *DirtyJSON) tokenize(s string) []token {
	var tokens []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '{':
			tokens = append(tokens, token{tokLBrace, "{"})
			i++
		case c == '}':
			tokens = append(tokens, token{tokRBrace, "}"})
			i++
		case c == '[':
			tokens = append(tokens, token{tokLBracket, "["})
			i++
		case c == ']':
			tokens = append(tokens, token{tokRBracket, "]"})
			i++
		case c == ':':
			tokens = append(tokens, token{tokColon, ":"})
			i++
		case c == ',':
			tokens = append(tokens, token{tokComma, ","})
			i++
		case c == '"':
			start := i
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			tokens = append(tokens, token{tokString, s[start:i]})
		case c == '-' || (c >= '0' && c <= '9'):
			start := i
			if c == '-' {
				i++
			}
			for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.' || s[i] == 'e' || s[i] == 'E' || s[i] == '+' || s[i] == '-') {
				i++
			}
			tokens = append(tokens, token{tokNumber, s[start:i]})
		default:
			// true / false / null or any other identifier
			start := i
			for i < len(s) && isUnquotedKeyChar(s[i]) {
				i++
			}
			tokens = append(tokens, token{tokBoolOrNull, s[start:i]})
		}
	}
	tokens = append(tokens, token{tokEOF, ""})
	return tokens
}

// ---------------------------------------------------------------------------
// Recursive-descent parser (tolerant)
// ---------------------------------------------------------------------------

func (d *DirtyJSON) parseValue(tokens []token, pos int) (any, int, error) {
	if pos >= len(tokens) {
		return nil, pos, fmt.Errorf("unexpected end of input")
	}
	tok := tokens[pos]
	switch tok.typ {
	case tokLBrace:
		return d.parseObject(tokens, pos)
	case tokLBracket:
		return d.parseArray(tokens, pos)
	case tokString:
		return unquote(tok.val), pos + 1, nil
	case tokNumber:
		return parseNumber(tok.val), pos + 1, nil
	case tokBoolOrNull:
		switch tok.val {
		case "true":
			return true, pos + 1, nil
		case "false":
			return false, pos + 1, nil
		case "null":
			return nil, pos + 1, nil
		default:
			return tok.val, pos + 1, nil // treat unknown as string
		}
	case tokRBrace, tokRBracket:
		// Missing value — return nil
		return nil, pos, nil
	case tokEOF:
		return nil, pos, fmt.Errorf("unexpected end of input")
	default:
		return nil, pos + 1, fmt.Errorf("unexpected token %q", tok.val)
	}
}

func (d *DirtyJSON) parseObject(tokens []token, pos int) (map[string]any, int, error) {
	obj := make(map[string]any)
	pos++ // skip {
	for pos < len(tokens) {
		tok := tokens[pos]
		if tok.typ == tokRBrace || tok.typ == tokEOF {
			pos++
			return obj, pos, nil
		}
		if tok.typ == tokComma {
			pos++
			continue
		}
		// Key
		if tok.typ != tokString {
			// Tolerate: skip unexpected token
			pos++
			continue
		}
		key := unquote(tok.val)
		pos++
		// Colon
		if pos < len(tokens) && tokens[pos].typ == tokColon {
			pos++
		}
		// Value
		val, next, err := d.parseValue(tokens, pos)
		if err != nil {
			return obj, next, err
		}
		obj[key] = val
		pos = next
		// Skip comma
		if pos < len(tokens) && tokens[pos].typ == tokComma {
			pos++
		}
	}
	return obj, pos, nil
}

func (d *DirtyJSON) parseArray(tokens []token, pos int) ([]any, int, error) {
	var arr []any
	pos++ // skip [
	for pos < len(tokens) {
		tok := tokens[pos]
		if tok.typ == tokRBracket || tok.typ == tokEOF {
			pos++
			return arr, pos, nil
		}
		if tok.typ == tokComma {
			pos++
			continue
		}
		val, next, err := d.parseValue(tokens, pos)
		if err != nil {
			return arr, next, err
		}
		arr = append(arr, val)
		pos = next
		if pos < len(tokens) && tokens[pos].typ == tokComma {
			pos++
		}
	}
	return arr, pos, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func parseNumber(s string) any {
	// Try int first.
	negative := false
	str := s
	if len(str) > 0 && str[0] == '-' {
		negative = true
		str = str[1:]
	}
	if !strings.ContainsAny(str, ".eE") {
		var n int64
		for i := 0; i < len(str); i++ {
			if str[i] < '0' || str[i] > '9' {
				goto asFloat
			}
			n = n*10 + int64(str[i]-'0')
		}
		if negative {
			return -n
		}
		return n
	}
asFloat:
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// detectCompletion checks whether the buffer looks like a complete JSON value.
func (d *DirtyJSON) detectCompletion() bool {
	s := strings.TrimSpace(string(d.buf))
	if s == "" {
		return false
	}
	// Primitive values: true, false, null, numbers, quoted strings
	if s == "true" || s == "false" || s == "null" {
		return true
	}
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return true
	}
	if unicode.IsDigit(rune(s[0])) || s[0] == '-' {
		allNum := true
		for _, c := range s[1:] {
			if !unicode.IsDigit(c) && c != '.' && c != 'e' && c != 'E' && c != '+' && c != '-' {
				allNum = false
				break
			}
		}
		if allNum {
			return true
		}
	}
	// Balanced braces
	depth := 0
	inStr := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' && (i == 0 || s[i-1] != '\\') {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch s[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}
	return depth <= 0
}
