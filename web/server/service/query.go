// # Grammar
// + Items beginning with underscore (_) are terminal.
// + Quoted items are literals.
// + Strings containing spaces must be delimeted with (single or double) quotes.
//   + Quotes contained in strings matching the delimeter must be escaped with a backslash (\)
//   + Backslashes contianed in a delimeted string must be escaped with a backslash (i.e. \\)
//
// query -> clause*
// clause -> [-]property_query | [-]tag_query | note_query | _string
// property_query -> "property:" _string [operator _string ]
// operator -> "=" | "!=" | "<" | "<=" | ">" | ">="
// tag_query -> "tag:" _string
// note_query -> "note:" _string
//
// ## Examples
// + **tag_query:** `tag:good`
// + **property_query:** `property:temperature`, `property:temperature>10`, `property:"annealing time"!=0`
// + **note_query:** `note:keyword`, `note:"important phrase"`
//
// ## TODO
// + parse property_query in a typed fashion
//   i.e. if the property is a boolean only allow `true` or `false` values with restricted operators

package service

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

type Position int

type DataQueryTokenKind int

const (
	EOF DataQueryTokenKind = iota
	ILLEGAL
	STRING_BARE
	STRING_DELIMITED
	KW_PROPERTY
	KW_TAG
	KW_NOTE
	NOT
	EQUAL
	NOT_EQUAL
	LESS
	LESS_EQUAL
	GREATER
	GREATER_EQUAL
)

var TOKENS = []string{
	EOF:              "EOF",
	ILLEGAL:          "ILLEGAL",
	STRING_BARE:      "STRING_BARE",
	STRING_DELIMITED: "STRING_DELIMITED",
	KW_PROPERTY:      "property:",
	KW_TAG:           "tag:",
	KW_NOTE:          "note:",
	NOT:              "-",
	EQUAL:            "=",
	NOT_EQUAL:        "!=",
	LESS:             "<",
	LESS_EQUAL:       "<=",
	GREATER:          ">",
	GREATER_EQUAL:    ">=",
}

func (t DataQueryTokenKind) String() string {
	return TOKENS[t]
}

type StringDelimeter rune

const (
	QUOTE_SINGLE StringDelimeter = '\''
	QUOTE_DOUBLE StringDelimeter = '"'
)

type Token struct {
	Pos  Position
	Kind DataQueryTokenKind
	Lit  string
}

func Tokenize(clause string) ([]Token, error) {
	lexer := NewDataQueryLexer(clause)
	var tokens []Token
	for {
		pos, kind, lit, err := lexer.NextToken()
		if err != nil {
			return tokens, err
		}
		if kind == EOF {
			return tokens, nil
		}

		tokens = append(tokens, Token{Pos: pos, Kind: kind, Lit: lit})
	}
}

type DataQueryLexer struct {
	pos      Position
	runes    []rune
	in_query bool
}

func NewDataQueryLexer(query string) *DataQueryLexer {
	return &DataQueryLexer{
		pos:      0,
		runes:    []rune(query),
		in_query: false,
	}
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

func (l *DataQueryLexer) is_valid_char(ch rune) bool {
	return (unicode.IsLetter(ch) ||
		unicode.IsDigit(ch) ||
		ch == '_' ||
		ch == '-' ||
		ch == ':' ||
		ch == '.')
}

func (l *DataQueryLexer) NextToken() (Position, DataQueryTokenKind, string, error) {
	for {
		start := l.pos
		next, valid := l.next()
		if !valid {
			l.in_query = false
			return l.pos, EOF, "", nil
		}

		switch next {
		case '-':
			l.in_query = false
			return start, NOT, "-", nil
		case '!':
			l.in_query = false
			_, match := l.next_if_eq('=')
			if match {
				return start, NOT_EQUAL, "!=", nil
			} else {
				return start, STRING_BARE, "!", nil
			}
		case '=':
			l.in_query = false
			return start, EQUAL, "=", nil
		case '>':
			l.in_query = false
			_, match := l.next_if_eq('=')
			if match {
				return start, GREATER_EQUAL, ">=", nil
			} else {
				return start, GREATER, ">", nil
			}
		case '<':
			l.in_query = false
			_, match := l.next_if_eq('=')
			if match {
				return start, LESS_EQUAL, "<=", nil
			} else {
				return start, LESS, "<", nil
			}
		case '\'', '"':
			l.in_query = false
			value := l.next_while(func(r rune) bool { return r != next })
			out := string(next) + string(value)
			final, match := l.next_if_eq(next)
			if match {
				out += string(final)
				return start, STRING_DELIMITED, out, nil
			} else {
				return start, STRING_DELIMITED, out, fmt.Errorf("unterminated string")
			}

		default:
			if unicode.IsSpace(next) {
				l.in_query = false
				continue
			} else if l.is_valid_char(next) {
				runes := l.next_while(func(r rune) bool { return l.is_valid_char(r) })
				value := string(next) + string(runes)
				if !l.in_query && strings.HasPrefix(value, KW_PROPERTY.String()) {
					l.pos = start + Position(len(KW_PROPERTY.String()))
					l.in_query = true
					return start, KW_PROPERTY, KW_PROPERTY.String(), nil
				} else if !l.in_query && strings.HasPrefix(value, KW_TAG.String()) {
					l.pos = start + Position(len(KW_TAG.String()))
					l.in_query = true
					return start, KW_TAG, KW_TAG.String(), nil
				} else if !l.in_query && strings.HasPrefix(value, KW_NOTE.String()) {
					l.pos = start + Position(len(KW_NOTE.String()))
					l.in_query = true
					return start, KW_NOTE, KW_NOTE.String(), nil
				} else {
					l.in_query = false
					return start, STRING_BARE, value, nil
				}
			} else {
				return start, ILLEGAL, string(next), fmt.Errorf("illegal character")
			}
		}
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

type DataQueryFilterKind string

const (
	DataQueryFilterKindProject  DataQueryFilterKind = "project"
	DataQueryFilterKindProperty DataQueryFilterKind = "property"
	DataQueryFilterKindTag      DataQueryFilterKind = "tag"
)

type DataQueryOperator string

const (
	DataQueryOperatorEqual          DataQueryOperator = "="
	DataQueryOperatorNotEqual       DataQueryOperator = "!="
	DataQueryOperatorLess           DataQueryOperator = "<"
	DataQueryOperatorLessOrEqual    DataQueryOperator = "<="
	DataQueryOperatorGreater        DataQueryOperator = ">"
	DataQueryOperatorGreaterOrEqual DataQueryOperator = ">="
)

type DataQueryFilter interface {
	Kind() DataQueryFilterKind
	Sql() string
}

type DataQueryFilterProject struct {
	Project uuid.UUID
}

func (f DataQueryFilterProject) Kind() DataQueryFilterKind {
	return DataQueryFilterKindProject
}

func (f DataQueryFilterProject) Sql() string {
	return `EXISTS (
		SELECT 1 FROM project_data_membership_ 
		WHERE _project=$1
	)`
}

type DataQueryFilterProperty struct {
	Key      string
	Operator *DataQueryFilterPropertyOperator
}

type DataQueryFilterPropertyOperator struct {
	Operator DataQueryOperator
	Kind     PropertyType
	Value    any
}

func (f DataQueryFilterProperty) Kind() DataQueryFilterKind {
	return DataQueryFilterKindProperty
}

func (f DataQueryFilterProperty) Sql() string {
	if f.Operator == nil {
		return `EXISTS (
			SELECT 1 FROM data_property_
			WHERE _key=$1
		)`
	} else {
		// TODO
		return ""
	}
}

type DataQueryFilterTag struct {
	Tag     string
	Project uuid.UUID
}

func (f DataQueryFilterTag) Kind() DataQueryFilterKind {
	return DataQueryFilterKindTag
}

func (f DataQueryFilterTag) Sql() string {
	if f.Project == uuid.Nil {
		return `EXISTS (
			SELECT 1 FROM project_data_tag_ 
			WHERE _tag=$1
		)`
	} else {
		return `EXISTS (
			SELECT 1 FROM project_data_tag_ 
			WHERE _project=$1 AND _tag=$2
		)`
	}
}

func ParseDataQueryClause(clause string) (DataQueryFilter, error) {
	panic("todo")
}
