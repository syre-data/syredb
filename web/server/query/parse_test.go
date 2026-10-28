package query

import (
	"testing"
)

func compareExprArrays(t *testing.T, expect []Expr, found []Expr) {
	t.Helper()
	if len(expect) != len(found) {
		t.Errorf("expression arrays have different length; expected %v, found %v", len(expect), len(found))
		return
	}

	for idx := range expect {
		compareExprs(t, expect[idx], found[idx])
	}
}

func compareExprs(t *testing.T, expect Expr, found Expr) {
	t.Helper()
	if expect.Kind != found.Kind {
		t.Errorf("expressions do not match; expected %v, but found %v", expect.Kind, found.Kind)
	}
	switch expect.Kind {
	case EXPR_EOF:
		panic("EOF should not be included in expression arrays")
	case EXPR_FULL_TEXT:
		e_val := expect.Expr.(string)
		f_val := found.Expr.(string)
		if e_val != f_val {
			t.Errorf("expression values do not match: expected %v, found %v", e_val, f_val)
		}
	case EXPR_NOT:
		e_val := expect.Expr.(Expr)
		f_val := found.Expr.(Expr)
		compareExprs(t, e_val, f_val)
	case EXPR_PROPERTY:
		e_val := expect.Expr.(ExprProperty)
		f_val := found.Expr.(ExprProperty)
		compareExprsProperty(t, e_val, f_val)
	case EXPR_TAG:
		e_val := expect.Expr.(string)
		f_val := found.Expr.(string)
		if e_val != f_val {
			t.Errorf("expression values do not match: expected %v, found %v", e_val, f_val)
		}
	case EXPR_NOTE:
		e_val := expect.Expr.(string)
		f_val := found.Expr.(string)
		if e_val != f_val {
			t.Errorf("expression values do not match: expected %v, found %v", e_val, f_val)
		}
	}
}

func compareExprsProperty(t *testing.T, expect ExprProperty, found ExprProperty) {
	if expect.Key != found.Key {
		t.Errorf("property keys do not match: expected %v, found %v", expect.Key, found.Key)
	}
	if expect.Op == nil && found.Op == nil {
		return
	}
	if expect.Op == nil && found.Op != nil {
		t.Errorf("property operators do not match: expected nil, found %v", found.Op.Op)
	}
	if expect.Op != nil && found.Op == nil {
		t.Errorf("property operators do not match: expected %v, found nil", expect.Op.Op)
	}

	e_op := expect.Op
	f_op := found.Op
	if e_op.Op != f_op.Op {
		t.Errorf("property operators do not match: expected %v, found %v", e_op.Op, f_op.Op)
	}
	if e_op.Value != f_op.Value {
		t.Errorf("property operator values do not match: expected %v, found %v", e_op.Value, f_op.Value)
	}
}

func TestParserEmpty(t *testing.T) {
	t1, err := tokenize("")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{}
	compareExprArrays(t, r1, p1)
}

func TestParserStringBare(t *testing.T) {
	t1, err := tokenize("test")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_FULL_TEXT,
			Expr: "test",
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("test phrase")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_FULL_TEXT,
			Expr: "test",
		},
		{
			Kind: EXPR_FULL_TEXT,
			Expr: "phrase",
		},
	}
	compareExprArrays(t, r2, p2)
}

func TestParserStringDelimited(t *testing.T) {
	t1, err := tokenize("'test'")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_FULL_TEXT,
			Expr: "test",
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("'test phrase'")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_FULL_TEXT,
			Expr: "test phrase",
		},
	}
	compareExprArrays(t, r2, p2)
}

func TestParserProperty(t *testing.T) {
	t1, err := tokenize("property:p1")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op:  nil,
			},
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("property:p1=10")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_EQUAL,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r2, p2)

	t3, err := tokenize("property:'test prop'")
	if err != nil {
		t.Error(err)
	}
	p3, err := parseTokens(t3)
	if err != nil {
		t.Error(err)
	}
	r3 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "test prop",
				Op:  nil,
			},
		},
	}
	compareExprArrays(t, r3, p3)

	t4, err := tokenize("property:'test prop'!=10")
	if err != nil {
		t.Error(err)
	}
	p4, err := parseTokens(t4)
	if err != nil {
		t.Error(err)
	}
	r4 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "test prop",
				Op: &ExprPropertyOperator{
					Op:    OP_NOT_EQUAL,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r4, p4)

	t5, err := tokenize("property:p1<10")
	if err != nil {
		t.Error(err)
	}
	p5, err := parseTokens(t5)
	if err != nil {
		t.Error(err)
	}
	r5 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_LESS,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r5, p5)

	t6, err := tokenize("property:p1<=10")
	if err != nil {
		t.Error(err)
	}
	p6, err := parseTokens(t6)
	if err != nil {
		t.Error(err)
	}
	r6 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_LESS_EQUAL,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r6, p6)

	t7, err := tokenize("property:p1>10")
	if err != nil {
		t.Error(err)
	}
	p7, err := parseTokens(t7)
	if err != nil {
		t.Error(err)
	}
	r7 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_GREATER,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r7, p7)

	t8, err := tokenize("property:p1>=10")
	if err != nil {
		t.Error(err)
	}
	p8, err := parseTokens(t8)
	if err != nil {
		t.Error(err)
	}
	r8 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_GREATER_EQUAL,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r8, p8)

	t9, err := tokenize("property:p1<10")
	if err != nil {
		t.Error(err)
	}
	p9, err := parseTokens(t9)
	if err != nil {
		t.Error(err)
	}
	r9 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_LESS,
					Value: "10",
				},
			},
		},
	}
	compareExprArrays(t, r9, p9)

	t10, err := tokenize("-property:p1<10")
	if err != nil {
		t.Error(err)
	}
	p10, err := parseTokens(t10)
	if err != nil {
		t.Error(err)
	}
	r10 := []Expr{
		{
			Kind: EXPR_NOT,
			Expr: Expr{
				Kind: EXPR_PROPERTY,
				Expr: ExprProperty{
					Key: "p1",
					Op: &ExprPropertyOperator{
						Op:    OP_LESS,
						Value: "10",
					},
				},
			},
		},
	}
	compareExprArrays(t, r10, p10)
}

func TestParserType(t *testing.T) {
	t1, err := tokenize("type:t1")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_TYPE,
			Expr: "t1",
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("type:'test type'")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_TYPE,
			Expr: "test type",
		},
	}
	compareExprArrays(t, r2, p2)
}

func TestParserTag(t *testing.T) {
	t1, err := tokenize("tag:t1")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_TAG,
			Expr: "t1",
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("tag:'test tag'")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_TAG,
			Expr: "test tag",
		},
	}
	compareExprArrays(t, r2, p2)
}

func TestParserNote(t *testing.T) {
	t1, err := tokenize("note:word")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_NOTE,
			Expr: "word",
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("note:'test phrase'")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_NOTE,
			Expr: "test phrase",
		},
	}
	compareExprArrays(t, r2, p2)
}

func TestParserComplex(t *testing.T) {
	t1, err := tokenize("property:p1>1 note:word")
	if err != nil {
		t.Error(err)
	}
	p1, err := parseTokens(t1)
	if err != nil {
		t.Error(err)
	}
	r1 := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "p1",
				Op: &ExprPropertyOperator{
					Op:    OP_GREATER,
					Value: "1",
				},
			},
		},
		{
			Kind: EXPR_NOTE,
			Expr: "word",
		},
	}
	compareExprArrays(t, r1, p1)

	t2, err := tokenize("-test")
	if err != nil {
		t.Error(err)
	}
	p2, err := parseTokens(t2)
	if err != nil {
		t.Error(err)
	}
	r2 := []Expr{
		{
			Kind: EXPR_NOT,
			Expr: Expr{
				Kind: EXPR_FULL_TEXT,
				Expr: "test",
			},
		},
	}
	compareExprArrays(t, r2, p2)

	t3, err := tokenize("=!=<><=>=")
	if err != nil {
		t.Error(err)
	}
	p3, err := parseTokens(t3)
	if err != nil {
		t.Error(err)
	}
	r3 := []Expr{
		{
			Kind: EXPR_FULL_TEXT,
			Expr: "=!=<><=>=",
		},
	}
	compareExprArrays(t, r3, p3)
}
