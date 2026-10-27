// See lex.go for grammar

package query

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type QueryParams struct {
	project uuid.UUID
	Sql     strings.Builder
	Args    []any
}

func BuildQuery(user uuid.UUID, exprs []Expr) QueryParams {
	q := QueryParams{
		project: uuid.Nil,
		Sql:     strings.Builder{},
		Args:    make([]any, 1, len(exprs)+1),
	}

	q.Sql.WriteString(
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1 
			AND (_permission='owner' OR _permission='read')
		)`,
	)
	q.Args[0] = user

	for _, expr := range exprs {
		q.Sql.WriteString(" AND ")
		q.AddExpr(expr, uuid.Nil)
	}

	return q
}

func BuildProjectQuery(user uuid.UUID, project uuid.UUID, exprs []Expr) QueryParams {
	q := QueryParams{
		project: project,
		Sql:     strings.Builder{},
		Args:    make([]any, 2, len(exprs)+2),
	}

	q.Sql.WriteString(
		`SELECT d._id from data_ as d WHERE EXISTS (
			SELECT 1 FROM data_user_permission_
			WHERE _data=d._id AND _user=$1 
			AND (_permission='owner' OR _permission='read')
		) AND EXISTS (
			SELECT 1 FROM project_data_membership_
			WHERE _project=$2 AND _data=d._id 
		)`,
	)
	q.Args[0] = user
	q.Args[1] = project

	for _, expr := range exprs {
		q.Sql.WriteString(" AND ")
		q.AddExpr(expr, project)
	}

	return q
}

func (q *QueryParams) AddExpr(expr Expr, project uuid.UUID) {
	switch expr.Kind {
	case EXPR_FULL_TEXT:
		panic("TODO")
	case EXPR_NOT:
		panic("TODO")
	case EXPR_PROPERTY:
		q.exprPropertyToSql(expr.Expr.(ExprProperty))
	case EXPR_TAG:
		q.exprTagToSql(expr.Expr.(string))
	case EXPR_NOTE:
		q.exprNoteToSql(expr.Expr.(string))
	case EXPR_EOF:
		panic("unreachable")
	default:
		panic(fmt.Sprintf("unexpected ExprKind: %#v", expr.Kind))
	}
}

func (q *QueryParams) exprPropertyToSql(expr ExprProperty) {
	if expr.Op == nil {
		// NB: Perhaps matching key here should use LIKE clause
		fmt.Fprintf(
			&q.Sql,
			`EXISTS (
				SELECT 1 FROM data_property_
				WHERE _data=d._id AND _key=$%d
			)`,
			len(q.Args)+1,
		)
		q.Args = append(q.Args, expr.Key)
	} else {
		panic("TODO")
	}
}

func (q *QueryParams) exprTagToSql(tag string) {
	// NB: Perhaps tag matching should use LIKE clause
	if q.project == uuid.Nil {
		fmt.Fprintf(
			&q.Sql,
			`EXISTS (
			SELECT 1 FROM project_data_tag_ 
			WHERE _data=d._id AND _tag=$%d
			)`,
			len(q.Args)+1,
		)
	} else {
		fmt.Fprintf(
			&q.Sql,
			`EXISTS (
			SELECT 1 FROM project_data_tag_ 
			WHERE _data=d._id AND _project=$2 AND _tag=$%d
			)`,
			len(q.Args)+1,
		)
	}
	q.Args = append(q.Args, tag)
}

func (q *QueryParams) exprNoteToSql(content string) {
	fmt.Fprintf(
		&q.Sql,
		`EXISTS (
		SELECT 1 FROM data_note_ 
		WHERE _data=d._id AND content LIKE $%d
		)`,
		len(q.Args)+1,
	)
	q.Args = append(q.Args, "%"+content+"%")
}
