// # Grammar
// + Items beginning with underscore (_) are terminal.
// + Quoted items are literals.
// + Strings containing spaces must be delimeted with (single or double) quotes.
//   + Quotes contained in strings matching the delimeter must be escaped with a backslash (\)
//   + Backslashes contianed in a delimeted string must be escaped with a backslash (i.e. \\)
//
// query -> clause*
// clause -> [-]type_query [-]property_query | [-]tag_query | note_query | _string
// type_query -> "type:" _string
// property_query -> "property:" _string [operator _string ]
// operator -> "=" | "!=" | "<" | "<=" | ">" | ">="
// tag_query -> "tag:" _string
// note_query -> "note:" _string
//
// ## Examples
// + **type_query:** `type:photoluminescence`, `-type:"impedence spectroscopy"`
// + **tag_query:** `tag:good`, `-tag:bad`
// + **property_query:** `property:temperature`, `property:temperature>10`, `property:"annealing time"!=0`, `-property:test=true`
// + **note_query:** `note:keyword`, `note:"important phrase"`
//
// ## TODO
// + `field` filter
// + `name` filter
// + `created` filter
// + `OR` groups
// + `project` filter
// + parse property_query in a typed fashion
//   i.e. if the property is a boolean only allow `true` or `false` values with restricted operators

package query

import (
	"fmt"
	"strings"
	"unicode"
)

type Position int

type DataQueryTokenKind int

const (
	TOKEN_EOF DataQueryTokenKind = iota
	TOKEN_ILLEGAL
	TOKEN_STRING_BARE
	TOKEN_STRING_DELIMITED
	TOKEN_KW_TYPE
	TOKEN_KW_PROPERTY
	TOKEN_KW_TAG
	TOKEN_KW_NOTE
	TOKEN_NOT
	TOKEN_OP_EQUAL
	TOKEN_OP_NOT_EQUAL
	TOKEN_OP_LESS
	TOKEN_OP_LESS_EQUAL
	TOKEN_OP_GREATER
	TOKEN_OP_GREATER_EQUAL
)

var TOKENS = []string{
	TOKEN_EOF:              "EOF",
	TOKEN_ILLEGAL:          "ILLEGAL",
	TOKEN_STRING_BARE:      "STRING_BARE",
	TOKEN_STRING_DELIMITED: "STRING_DELIMITED",
	TOKEN_KW_TYPE:          "type:",
	TOKEN_KW_PROPERTY:      "property:",
	TOKEN_KW_TAG:           "tag:",
	TOKEN_KW_NOTE:          "note:",
	TOKEN_NOT:              "-",
	TOKEN_OP_EQUAL:         "=",
	TOKEN_OP_NOT_EQUAL:     "!=",
	TOKEN_OP_LESS:          "<",
	TOKEN_OP_LESS_EQUAL:    "<=",
	TOKEN_OP_GREATER:       ">",
	TOKEN_OP_GREATER_EQUAL: ">=",
}

func (t DataQueryTokenKind) String() string {
	return TOKENS[t]
}

type Token struct {
	Pos  Position
	Kind DataQueryTokenKind
	Lit  string
}

func tokenize(clause string) ([]Token, error) {
	lexer := NewDataQueryLexer(clause)
	var tokens []Token
	for {
		pos, kind, lit, err := lexer.NextToken()
		if err != nil {
			return tokens, err
		}

		tokens = append(
			tokens,
			Token{Pos: pos, Kind: kind, Lit: lit},
		)
		if kind == TOKEN_EOF {
			return tokens, nil
		}
	}
}

type DataQueryLexer struct {
	pos               Position
	runes             []rune
	in_property_query bool
}

func NewDataQueryLexer(query string) *DataQueryLexer {
	return &DataQueryLexer{
		pos:               0,
		runes:             []rune(query),
		in_property_query: false,
	}
}

// Returns the rune ahead letters ahead without consuming it, if avialable.
// Second return value indicates if the returned rune is valid or not
// (i.e. is false if the lexer has reached the end of its input)
func (l *DataQueryLexer) peek_ahead(ahead int) (rune, bool) {
	if int(l.pos) > len(l.runes) {
		panic("invalid position")
	}

	next := int(l.pos) + ahead - 1
	if next >= len(l.runes) {
		return rune(0), false
	}

	return l.runes[next], true
}

// Returns the next rune without consuming it, if avialable.
// Second return value indicates if the returned rune is valid or not
// (i.e. is false if the lexer has reached the end of its input)
func (l *DataQueryLexer) peek() (rune, bool) {
	if int(l.pos) > len(l.runes) {
		panic("invalid position")
	}

	if int(l.pos) == len(l.runes) {
		return rune(0), false
	}

	return l.runes[l.pos], true
}

// Consume the next rune.
// Second return value indicates if the returned rune is valid or not
// (i.e. is false if the lexer has reached the end of its input)
func (l *DataQueryLexer) next() (rune, bool) {
	if int(l.pos) > len(l.runes) {
		panic("invalid position")
	}

	if int(l.pos) == len(l.runes) {
		return rune(0), false
	}

	r := l.runes[l.pos]
	l.pos += 1
	return r, true
}

// Consume the next rune if it is equal to expected.
// Second return value indicates if the returned rune is valid or not
// (i.e. is false if the lexer has reached the end of its input,
// or the next rune does not match expected)
func (l *DataQueryLexer) next_if_eq(expected rune) (rune, bool) {
	next, valid := l.peek()
	if valid && next == expected {
		l.pos += 1
		return next, true
	} else {
		return rune(0), false
	}
}

// Consume the next rune if it is equal to expected and return the matched Token.
// If the next rune is not equal to expected, do not consume it and reutrn unmatched.
func (l *DataQueryLexer) next_if_else(expected rune, matched Token, unmatched Token) Token {
	_, match := l.next_if_eq(expected)
	if match {
		return matched
	} else {
		return unmatched
	}
}

// Consume characters while predicate remains true.
func (l *DataQueryLexer) next_while(predicate func(rune) bool) []rune {
	var consumed []rune
	for {
		next, valid := l.peek()
		if !valid {
			return consumed
		}
		if predicate(next) {
			next, valid := l.next()
			if !valid {
				panic("next rune invalid")
			}

			consumed = append(consumed, next)
		} else {
			return consumed
		}
	}
}

func (l *DataQueryLexer) NextToken() (Position, DataQueryTokenKind, string, error) {
	spaces := l.next_while(func(r rune) bool { return unicode.IsSpace(r) })
	if len(spaces) > 0 {
		l.in_property_query = false
	}

	start := l.pos
	next, valid := l.next()
	if !valid {
		l.in_property_query = false
		return l.pos, TOKEN_EOF, "", nil
	}

	if l.in_property_query {
		switch next {
		case '!':
			_, match := l.next_if_eq('=')
			if match {
				return start, TOKEN_OP_NOT_EQUAL, "!=", nil
			}
		case '=':
			return start, TOKEN_OP_EQUAL, "=", nil
		case '>':
			_, match := l.next_if_eq('=')
			if match {
				return start, TOKEN_OP_GREATER_EQUAL, ">=", nil
			} else {
				return start, TOKEN_OP_GREATER, ">", nil
			}
		case '<':
			_, match := l.next_if_eq('=')
			if match {
				return start, TOKEN_OP_LESS_EQUAL, "<=", nil
			} else {
				return start, TOKEN_OP_LESS, "<", nil
			}
		}
	} else {
		switch next {
		case '-':
			l.in_property_query = false
			return start, TOKEN_NOT, "-", nil
		}
	}

	if next == '\'' || next == '"' {
		value := l.next_while(func(r rune) bool { return r != next })
		out := string(next) + string(value)
		final, match := l.next_if_eq(next)
		if match {
			out += string(final)
			return start, TOKEN_STRING_DELIMITED, out, nil
		} else {
			return start, TOKEN_STRING_DELIMITED, out, fmt.Errorf("unterminated string")
		}
	}

	var runes []rune
	if l.in_property_query {
		for {
			r, valid := l.peek()
			if !valid {
				break
			}
			if unicode.IsSpace(r) {
				break
			}
			if r == '=' || r == '>' || r == '<' {
				break
			}
			if r == '!' {
				rn, valid := l.peek_ahead(2)
				if !valid || rn == '=' {
					break
				}
			}

			r, valid = l.next()
			if !valid {
				panic("invalid character")
			}
			runes = append(runes, r)
		}
	} else {
		runes = l.next_while(func(r rune) bool { return !unicode.IsSpace(r) })
	}
	value := string(next) + string(runes)
	if !l.in_property_query {
		if strings.HasPrefix(value, TOKEN_KW_TYPE.String()) {
			l.pos = start + Position(len(TOKEN_KW_TYPE.String()))
			return start, TOKEN_KW_TYPE, TOKEN_KW_TYPE.String(), nil
		}
		if strings.HasPrefix(value, TOKEN_KW_PROPERTY.String()) {
			l.in_property_query = true
			l.pos = start + Position(len(TOKEN_KW_PROPERTY.String()))
			return start, TOKEN_KW_PROPERTY, TOKEN_KW_PROPERTY.String(), nil
		}
		if strings.HasPrefix(value, TOKEN_KW_TAG.String()) {
			l.pos = start + Position(len(TOKEN_KW_TAG.String()))
			return start, TOKEN_KW_TAG, TOKEN_KW_TAG.String(), nil
		}
		if strings.HasPrefix(value, TOKEN_KW_NOTE.String()) {
			l.pos = start + Position(len(TOKEN_KW_NOTE.String()))
			return start, TOKEN_KW_NOTE, TOKEN_KW_NOTE.String(), nil
		}
	}

	return start, TOKEN_STRING_BARE, value, nil
}
