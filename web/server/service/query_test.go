package service

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

func TestLexerStringBare(t *testing.T) {
	t1, err := Tokenize("test_string")
	r1 := []Token{
		{Pos: 0, Kind: STRING_BARE, Lit: "test_string"},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := Tokenize("1one")
	r2 := []Token{
		{Pos: 0, Kind: STRING_BARE, Lit: "1one"},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)

	t3, err := Tokenize("_lot-of.marks")
	r3 := []Token{
		{Pos: 0, Kind: STRING_BARE, Lit: "_lot-of.marks"},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r3, t3)
}

func TestLexerStringDelimeted(t *testing.T) {
	t1, err := Tokenize("'test string'")
	r1 := []Token{
		{Pos: 0, Kind: STRING_DELIMITED, Lit: "'test string'"},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)
}

func TestLexerStringDelimetedErr(t *testing.T) {
	_, e1 := Tokenize("'test")
	if e1 == nil {
		t.Errorf("expected error")
	}

	_, e2 := Tokenize("\"test string")
	if e2 == nil {
		t.Errorf("expected error")
	}
}

func TestLexerKeywords(t *testing.T) {
	t1, err := Tokenize("property:p1")
	r1 := []Token{
		{Pos: 0, Kind: KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: STRING_BARE, Lit: "p1"},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r1, t1)

	t2, err := Tokenize("property:p1=1")
	r2 := []Token{
		{Pos: 0, Kind: KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: STRING_BARE, Lit: "p1"},
		{Pos: 11, Kind: EQUAL, Lit: "="},
		{Pos: 12, Kind: STRING_BARE, Lit: "1"},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r2, t2)

	t3, err := Tokenize("property:\"test prop\"")
	r3 := []Token{
		{Pos: 0, Kind: KW_PROPERTY, Lit: "property:"},
		{Pos: 9, Kind: STRING_DELIMITED, Lit: "\"test prop\""},
	}

	if err != nil {
		t.Error(err)
	}
	compareTokenStream(t, r3, t3)
}
