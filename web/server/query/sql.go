// See lex.go for grammar

package query

import (
	"fmt"
	"strings"
	"unicode"

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
		q.exprFullTextToSql(expr.Expr.(string))
	case EXPR_NOT:
		panic("TODO")
	case EXPR_PROPERTY:
		q.exprPropertyToSql(expr.Expr.(ExprProperty))
	case EXPR_TYPE:
		q.exprTypeToSql(expr.Expr.(string))
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

func (q *QueryParams) exprFullTextToSql(expr string) {
	// `d._id` is id of current data
	op_like := "ILIKE"
	for _, r := range expr {
		if unicode.IsUpper(r) {
			op_like = "LIKE"
			break
		}
	}

	q.Args = append(q.Args, expr+"%")
	idx_arg_wildcard_trailing := len(q.Args)
	q.Args = append(q.Args, "%"+expr+"%")
	idx_arg_wildcard_surrounded := len(q.Args)
	subexprs := make([]string, 3)
	subexprs[0] = fmt.Sprintf( // property keys
		`EXISTS (
			SELECT 1 FROM data_property_ 
			WHERE _data=d._id AND _key %s $%d
		)`,
		op_like,
		idx_arg_wildcard_trailing,
	)
	subexprs[1] = fmt.Sprintf( // data type
		`EXISTS (
			SELECT 1 FROM data_ JOIN data_type_ 
			ON data_._type=data_type_._id
			WHERE data_._id=d._id AND data_type_.label %s $%d
		)`,
		op_like,
		idx_arg_wildcard_trailing,
	)
	subexprs[2] = fmt.Sprintf( // creator - user
		`EXISTS (
			SELECT 1 FROM data_ 
			JOIN data_creator_user_ c ON data_._id=c._data
			JOIN user_ u ON c._creator=u._id
			JOIN data_origin_ o ON c._origin=o._id
			WHERE data_._id=d._id AND (
				u.email %[1]s $%[2]d
				OR u.name %[1]s $%[3]d
				OR o.label %[1]s $%[2]d
			)
		)`,
		op_like,
		idx_arg_wildcard_trailing,
		idx_arg_wildcard_surrounded,
	)

	q.Sql.WriteString("(")
	q.Sql.WriteString(strings.Join(subexprs, " OR "))
	q.Sql.WriteString(")")
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
		fmt.Fprintf(
			&q.Sql,
			`EXISTS (
				SELECT 1 FROM data_property_
				WHERE _data=d._id AND _key=$%d AND value%v$%d
			)`,
			len(q.Args)+1,
			expr.Op.Op.String(),
			len(q.Args)+2,
		)
		q.Args = append(q.Args, expr.Key)
		q.Args = append(q.Args, expr.Op.Value)
	}
}

func (q *QueryParams) exprTypeToSql(data_type string) {
	fmt.Fprintf(
		&q.Sql,
		`EXISTS (
			SELECT 1 FROM data_ JOIN data_type_ ON data_._type=data_type_._id
			WHERE data_._id=d._id AND data_type_.label=$%d
		)`,
		len(q.Args)+1,
	)
	q.Args = append(q.Args, data_type)
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
