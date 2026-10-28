package query

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func compareSql(t *testing.T, expect string, sql string) bool {
	t.Helper()
	e := strings.Fields(expect)
	s := strings.Fields(sql)
	if len(e) != len(s) {
		t.Errorf(
			"sql does not match: expected\n%v\nfound\n%v",
			strings.Join(e, " "),
			strings.Join(s, " "),
		)
		return false
	}
	for idx := range e {
		if e[idx] != s[idx] {
			t.Errorf(
				"sql does not match at %v: expected %v, found %v\n\nexpected\n%v\n\nfound\n%v",
				idx+1,
				e[idx],
				s[idx],
				strings.Join(e, " "),
				strings.Join(s, " "),
			)
			return false
		}
	}

	return true
}

func compareQuery(t *testing.T, sql string, args []any, q QueryParams) {
	t.Helper()
	sqlValid := compareSql(t, sql, q.Sql.String())
	if !sqlValid {
		return
	}

	if !cmp.Equal(args, q.Args) {
		t.Errorf("args do not match: exptected \n%v\nfound\n%v", args, q.Args)
	}
}

func TestQueryEmpty(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	q := BuildQuery(user, []Expr{})
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		)`
	args := []any{user}
	compareQuery(t, sql, args, q)
}

func TestProjectQueryEmpty(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	project, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	q := BuildProjectQuery(user, project, []Expr{})
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_membership_
			WHERE _project=$2 AND _data=d._id 
		)`
	args := []any{user, project}
	compareQuery(t, sql, args, q)
}

func TestQueryProperty(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "test_key",
				Op:  nil,
			},
		},
	}
	q := BuildQuery(user, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM data_property_
			WHERE _data=d._id AND _key=$2
		)`
	args := []any{user, "test_key"}
	compareQuery(t, sql, args, q)
}

func TestProjectQueryProperty(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	project, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_PROPERTY,
			Expr: ExprProperty{
				Key: "test_key",
				Op:  nil,
			},
		},
	}
	q := BuildProjectQuery(user, project, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_membership_
			WHERE _project=$2 AND _data=d._id 
		) AND EXISTS (
			SELECT 1 FROM data_property_
			WHERE _data=d._id AND _key=$3
		)`
	args := []any{user, project, "test_key"}
	compareQuery(t, sql, args, q)
}

func TestQueryType(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_TYPE,
			Expr: "test_type",
		},
	}
	q := BuildQuery(user, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM data_ JOIN data_type_ ON data_._type=data_type_._id
			WHERE data_._id=d._id AND data_type_.label=$2
		)`
	args := []any{user, "test_type"}
	compareQuery(t, sql, args, q)
}

func TestProjectQueryType(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	project, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_TYPE,
			Expr: "test_type",
		},
	}
	q := BuildProjectQuery(user, project, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_membership_
			WHERE _project=$2 AND _data=d._id 
		) AND EXISTS (
			SELECT 1 FROM data_ JOIN data_type_ ON data_._type=data_type_._id
			WHERE data_._id=d._id AND data_type_.label=$3
		)`
	args := []any{user, project, "test_type"}
	compareQuery(t, sql, args, q)
}

func TestQueryTag(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_TAG,
			Expr: "test_tag",
		},
	}
	q := BuildQuery(user, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_tag_
			WHERE _data=d._id AND _tag=$2
		)`
	args := []any{user, "test_tag"}
	compareQuery(t, sql, args, q)
}

func TestProjectQueryTag(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	project, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_TAG,
			Expr: "test_tag",
		},
	}
	q := BuildProjectQuery(user, project, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_membership_
			WHERE _project=$2 AND _data=d._id 
		) AND EXISTS (
			SELECT 1 FROM project_data_tag_
			WHERE _data=d._id AND _project=$2 AND _tag=$3
		)`
	args := []any{user, project, "test_tag"}
	compareQuery(t, sql, args, q)
}

func TestQueryNote(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_NOTE,
			Expr: "word",
		},
	}
	q := BuildQuery(user, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM data_note_
			WHERE _data=d._id AND content LIKE $2
		)`
	args := []any{user, "%word%"}
	compareQuery(t, sql, args, q)
}

func TestProjectQueryNote(t *testing.T) {
	user, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	project, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	exprs := []Expr{
		{
			Kind: EXPR_NOTE,
			Expr: "word",
		},
	}
	q := BuildProjectQuery(user, project, exprs)
	sql :=
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_membership_
			WHERE _project=$2 AND _data=d._id 
		) AND EXISTS (
			SELECT 1 FROM data_note_
			WHERE _data=d._id AND content LIKE $3
		)`
	args := []any{user, project, "%word%"}
	compareQuery(t, sql, args, q)
}
