package query

import (
	"testing"
)

func compareTokenStream(t *testing.T, expect []Token, found []Token) {
	t.Helper()
	if len(expect) != len(found) {
		t.Errorf("token streams have different length; expected %v, found %v", len(expect), len(found))
		return
	}

	for idx := range expect {
		compareTokens(t, expect[idx], found[idx])
	}
}

func compareTokens(t *testing.T, expect Token, found Token) {
	t.Helper()
	if expect.Kind != found.Kind {
		t.Errorf("token kinds do not match; expected %v, but found %v", expect.Kind, found.Kind)
	}
	if expect.Lit != found.Lit {
		t.Errorf("token values do not match; expected %v, but found %v", expect.Lit, found.Lit)
	}
	if expect.Pos != found.Pos {
		t.Errorf("token positions do not match; expected %v, but found %v", expect.Pos, found.Pos)
	}
}

func TestLexerEmpty(t *testing.T) {
	t1, err := tokenize("")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)
}

func TestLexerStringBare(t *testing.T) {
	t1, err := tokenize("test_string")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_STRING_BARE, Lit: "test_string"},
		{Pos: 11, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := tokenize("1one")
	r2 := []Token{
		{Pos: 0, Kind: TOKEN_STRING_BARE, Lit: "1one"},
		{Pos: 4, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)

	t3, err := tokenize("_lot-of.marks")
	r3 := []Token{
		{Pos: 0, Kind: TOKEN_STRING_BARE, Lit: "_lot-of.marks"},
		{Pos: 13, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r3, t3)
}

func TestLexerStringDelimeted(t *testing.T) {
	t1, err := tokenize("'test string'")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_STRING_DELIMITED, Lit: "'test string'"},
		{Pos: 13, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)
}

func TestLexerStringDelimetedErr(t *testing.T) {
	_, e1 := tokenize("'test")
	if e1 == nil {
		t.Errorf("expected error")
	}

	_, e2 := tokenize("\"test string")
	if e2 == nil {
		t.Errorf("expected error")
	}
}

func TestLexerKeywordProperty(t *testing.T) {
	t1, err := tokenize("property:p1")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := tokenize("property:p1=1")
	r2 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: TOKEN_OP_EQUAL, Lit: "="},
		{Pos: 12, Kind: TOKEN_STRING_BARE, Lit: "1"},
		{Pos: 13, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)

	t3, err := tokenize("property:\"test prop\"")
	r3 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_DELIMITED, Lit: "\"test prop\""},
		{Pos: 20, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r3, t3)

	t4, err := tokenize("property:\"test prop\"!=10")
	r4 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_DELIMITED, Lit: "\"test prop\""},
		{Pos: 20, Kind: TOKEN_OP_NOT_EQUAL, Lit: "!="},
		{Pos: 22, Kind: TOKEN_STRING_BARE, Lit: "10"},
		{Pos: 24, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r4, t4)

	t5, err := tokenize("property:p1<10")
	r5 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: TOKEN_OP_LESS, Lit: "<"},
		{Pos: 12, Kind: TOKEN_STRING_BARE, Lit: "10"},
		{Pos: 14, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r5, t5)

	t6, err := tokenize("property:p1<=10")
	r6 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: TOKEN_OP_LESS_EQUAL, Lit: "<="},
		{Pos: 13, Kind: TOKEN_STRING_BARE, Lit: "10"},
		{Pos: 15, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r6, t6)

	t7, err := tokenize("property:p1>10")
	r7 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: TOKEN_OP_GREATER, Lit: ">"},
		{Pos: 12, Kind: TOKEN_STRING_BARE, Lit: "10"},
		{Pos: 14, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r7, t7)

	t8, err := tokenize("property:p1>=10")
	r8 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: TOKEN_OP_GREATER_EQUAL, Lit: ">="},
		{Pos: 13, Kind: TOKEN_STRING_BARE, Lit: "10"},
		{Pos: 15, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r8, t8)

	t9, err := tokenize("-property:p1>=10")
	r9 := []Token{
		{Pos: 0, Kind: TOKEN_NOT, Lit: "-"},
		{Pos: 1, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 10, Kind: TOKEN_STRING_BARE, Lit: "p1"},
		{Pos: 12, Kind: TOKEN_OP_GREATER_EQUAL, Lit: ">="},
		{Pos: 14, Kind: TOKEN_STRING_BARE, Lit: "10"},
		{Pos: 16, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r9, t9)
}

func TestLexerKeywordTag(t *testing.T) {
	t1, err := tokenize("tag:t1")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_KW_TAG, Lit: "tag:"},
		{Pos: 4, Kind: TOKEN_STRING_BARE, Lit: "t1"},
		{Pos: 6, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := tokenize("tag:\"test tag\"")
	r2 := []Token{
		{Pos: 0, Kind: TOKEN_KW_TAG, Lit: "tag:"},
		{Pos: 4, Kind: TOKEN_STRING_DELIMITED, Lit: "\"test tag\""},
		{Pos: 14, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)

	t3, err := tokenize("-tag:t1")
	r3 := []Token{
		{Pos: 0, Kind: TOKEN_NOT, Lit: "-"},
		{Pos: 1, Kind: TOKEN_KW_TAG, Lit: "tag:"},
		{Pos: 5, Kind: TOKEN_STRING_BARE, Lit: "t1"},
		{Pos: 7, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r3, t3)
}

func TestLexerKeywordNote(t *testing.T) {
	t1, err := tokenize("note:word")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_KW_NOTE, Lit: "note:"},
		{Pos: 5, Kind: TOKEN_STRING_BARE, Lit: "word"},
		{Pos: 9, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := tokenize("note:\"note phrase\"")
	r2 := []Token{
		{Pos: 0, Kind: TOKEN_KW_NOTE, Lit: "note:"},
		{Pos: 5, Kind: TOKEN_STRING_DELIMITED, Lit: "\"note phrase\""},
		{Pos: 18, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)
}

func TestLexerComplex(t *testing.T) {
	t1, err := tokenize("property:t1>1 note:word")
	r1 := []Token{
		{Pos: 0, Kind: TOKEN_KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: TOKEN_STRING_BARE, Lit: "t1"},
		{Pos: 11, Kind: TOKEN_OP_GREATER, Lit: ">"},
		{Pos: 12, Kind: TOKEN_STRING_BARE, Lit: "1"},
		{Pos: 14, Kind: TOKEN_KW_NOTE, Lit: "note:"},
		{Pos: 19, Kind: TOKEN_STRING_BARE, Lit: "word"},
		{Pos: 23, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := tokenize("-test")
	r2 := []Token{
		{Pos: 0, Kind: TOKEN_NOT, Lit: "-"},
		{Pos: 1, Kind: TOKEN_STRING_BARE, Lit: "test"},
		{Pos: 5, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)

	t3, err := tokenize("=!=<><=>=")
	r3 := []Token{
		{Pos: 0, Kind: TOKEN_STRING_BARE, Lit: "=!=<><=>="},
		{Pos: 9, Kind: TOKEN_EOF, Lit: ""},
	}
	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r3, t3)
}
