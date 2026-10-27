// See lex.go for grammar

package query

import (
	"fmt"
	"slices"
)

type ExprKind int

const (
	EXPR_EOF ExprKind = iota
	EXPR_FULL_TEXT
	EXPR_NOT
	EXPR_PROPERTY
	EXPR_TAG
	EXPR_NOTE
)

type Expr struct {
	Kind ExprKind
	Expr any
}

type ExprUnrestriced struct {
	Value string
}

type ExprProperty struct {
	Key string
	Op  *ExprPropertyOperator
}

type ExprPropertyOperatorKind int

const (
	OP_EQUAL ExprPropertyOperatorKind = iota
	OP_NOT_EQUAL
	OP_GREATER
	OP_GREATER_EQUAL
	OP_LESS
	OP_LESS_EQUAL
)

type ExprPropertyOperator struct {
	Op    ExprPropertyOperatorKind
	Value string
}

type Parser struct {
	tokens []Token
	cursor int
}

func NewParser(tokens []Token) Parser {
	return Parser{tokens: tokens, cursor: 0}
}

// Returns the next token if valid.
// Second return value indicates token validity
// (i.e. if any tokens remain).
func (p *Parser) peek() (Token, bool) {
	if p.cursor > len(p.tokens) {
		panic("invalid cursor")
	}

	if p.cursor == len(p.tokens) {
		return Token{}, false
	}

	return p.tokens[p.cursor], true
}

// Consumes the next token if valid.
// Second return value indicates token validity
// (i.e. if any tokens remain).
func (p *Parser) next() (Token, bool) {
	if p.cursor > len(p.tokens) {
		panic("invalid cursor")
	}

	if p.cursor == len(p.tokens) {
		return Token{}, false
	}

	tok := p.tokens[p.cursor]
	p.cursor += 1
	return tok, true
}

func (p *Parser) NextExpr() (Expr, error) {
	return p.parsePrefix()
}

func (p *Parser) parsePrefix() (Expr, error) {
	next, valid := p.peek()
	if !valid {
		panic("token stream ended unexpectedly")
	}

	switch next.Kind {
	case TOKEN_EOF:
		return Expr{Kind: EXPR_EOF, Expr: nil}, nil
	case TOKEN_NOT,
		TOKEN_KW_NOTE,
		TOKEN_KW_PROPERTY,
		TOKEN_KW_TAG,
		TOKEN_STRING_BARE,
		TOKEN_STRING_DELIMITED:
		return p.parseUnary()
	case TOKEN_OP_EQUAL,
		TOKEN_OP_GREATER,
		TOKEN_OP_GREATER_EQUAL,
		TOKEN_ILLEGAL,
		TOKEN_OP_LESS,
		TOKEN_OP_LESS_EQUAL,
		TOKEN_OP_NOT_EQUAL:
		return Expr{}, fmt.Errorf("invalid prefix: %#v", next)
	default:
		panic(fmt.Sprintf("unexpected service.DataQueryTokenKind: %#v", next.Kind))
	}
}

func (p *Parser) parseUnary() (Expr, error) {
	next, valid := p.next()
	if !valid {
		panic("token stream unexpectedly empty")
	}

	switch next.Kind {
	case TOKEN_NOT:
		expr, err := p.NextExpr()
		if err != nil {
			return Expr{}, err
		}
		return Expr{
			Kind: EXPR_NOT,
			Expr: expr,
		}, nil
	case TOKEN_KW_PROPERTY:
		return p.parsePropertyQuery()
	case TOKEN_KW_TAG:
		return p.parseTagQuery()
	case TOKEN_KW_NOTE:
		return p.parseNoteQuery()
	case TOKEN_STRING_BARE:
		return Expr{
			Kind: EXPR_FULL_TEXT,
			Expr: next.Lit,
		}, nil
	case TOKEN_STRING_DELIMITED:
		return Expr{
			Kind: EXPR_FULL_TEXT,
			Expr: next.Lit[1 : len(next.Lit)-1],
		}, nil
	default:
		panic("unreachable")
	}
}

func (p *Parser) parsePropertyQuery() (Expr, error) {
	VALID_OPS := []DataQueryTokenKind{
		TOKEN_OP_EQUAL,
		TOKEN_OP_NOT_EQUAL,
		TOKEN_OP_GREATER,
		TOKEN_OP_GREATER_EQUAL,
		TOKEN_OP_LESS,
		TOKEN_OP_LESS_EQUAL,
	}

	next, valid := p.next()
	if !valid {
		panic("unexpected end of input")
	}

	var key string
	switch next.Kind {
	case TOKEN_STRING_BARE:
		key = next.Lit
	case TOKEN_STRING_DELIMITED:
		key = next.Lit[1 : len(next.Lit)-1]
	default:
		return Expr{}, fmt.Errorf("invalid property query")
	}

	next, valid = p.peek()
	if !valid {
		panic("unexpected end of input")
	}

	if !slices.Contains(VALID_OPS, next.Kind) {
		return Expr{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: key,
				Op:  nil,
			},
		}, nil
	}

	next, valid = p.next()
	if !valid {
		panic("unexpected end of input")
	}
	var op ExprPropertyOperatorKind
	switch next.Kind {
	case TOKEN_OP_EQUAL:
		op = OP_EQUAL
	case TOKEN_OP_NOT_EQUAL:
		op = OP_NOT_EQUAL
	case TOKEN_OP_GREATER:
		op = OP_GREATER
	case TOKEN_OP_GREATER_EQUAL:
		op = OP_GREATER_EQUAL
	case TOKEN_OP_LESS:
		op = OP_LESS
	case TOKEN_OP_LESS_EQUAL:
		op = OP_LESS_EQUAL
	default:
		panic("unreachable")
	}

	next, valid = p.next()
	if !valid {
		panic("unexpected end of input")
	}

	var filter string
	switch next.Kind {
	case TOKEN_STRING_BARE:
		filter = next.Lit
	case TOKEN_STRING_DELIMITED:
		filter = next.Lit[1 : len(next.Lit)-1]
	default:
		return Expr{}, fmt.Errorf("invalid property query")
	}

	return Expr{
		Kind: EXPR_PROPERTY,
		Expr: ExprProperty{
			Key: key,
			Op: &ExprPropertyOperator{
				Op:    op,
				Value: filter,
			},
		},
	}, nil
}

func (p *Parser) parseTagQuery() (Expr, error) {
	next, valid := p.next()
	if !valid {
		panic("unexpected end of input")
	}

	var tag string
	switch next.Kind {
	case TOKEN_STRING_BARE:
		tag = next.Lit
	case TOKEN_STRING_DELIMITED:
		tag = next.Lit[1 : len(next.Lit)-1]
	default:
		return Expr{}, fmt.Errorf("invalid tag query")
	}

	return Expr{
		Kind: EXPR_TAG,
		Expr: tag,
	}, nil
}

func (p *Parser) parseNoteQuery() (Expr, error) {
	next, valid := p.next()
	if !valid {
		panic("unexpected end of input")
	}

	var filter string
	switch next.Kind {
	case TOKEN_STRING_BARE:
		filter = next.Lit
	case TOKEN_STRING_DELIMITED:
		filter = next.Lit[1 : len(next.Lit)-1]
	default:
		return Expr{}, fmt.Errorf("invalid note query")
	}

	return Expr{
		Kind: EXPR_NOTE,
		Expr: filter,
	}, nil
}

func parseTokens(tokens []Token) ([]Expr, error) {
	parser := NewParser(tokens)
	var exprs []Expr
	for {
		expr, err := parser.NextExpr()
		if err != nil {
			return exprs, err
		}
		if expr.Kind == EXPR_EOF {
			return exprs, nil
		}

		exprs = append(exprs, expr)
	}
}

func Parse(input string) ([]Expr, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	return parseTokens(tokens)
}
