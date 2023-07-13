package cloudmon

import (
	"encoding/json"
	"regexp"
	"strings"
)

type LogEntry map[string]any

func newFieldLoader(prefix, name string) func(e LogEntry) any {
	parts := strings.Split(name, ".")
	return func(e LogEntry) any {
		var m map[string]any = e
		for i := 0; i < len(parts)-1; i++ {
			mi, ok := e[parts[i]]
			if !ok {
				return nil
			}
			if m, ok = mi.(map[string]any); !ok {
				return nil
			}
		}
		return m[parts[len(parts)-1]]
	}
}

func numberToFloat64(v any) (float64, bool) {
	lhsi, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	lhs, err := lhsi.Float64()
	return lhs, err == nil
}

type queryFunc func(e LogEntry) bool
type exprFunc func(v any) bool

func newQuery(ast *Query) queryFunc {
	var q queryFunc = newTerm(ast.First)
	for i := len(ast.Rest) - 1; i >= 0; i-- {
		q = newComplexQuery(ast.Rest[i], q)
	}
	return q
}

func newSubQuery(ast *SubQuery) queryFunc {
	q := newQuery(ast.Query)
	if ast.Exclusion {
		return func(e LogEntry) bool { return !q(e) }
	}
	return q
}

func newTerm(ast *Term) queryFunc {
	switch {
	case ast.Condition != nil:
		return newCondition(ast.Condition)
	case ast.FreeCondition != nil:
		return newFreeCondition(ast.FreeCondition)
	case ast.SubQuery != nil:
		return newSubQuery(ast.SubQuery)
	default:
		panic("invalid term")
	}
}

func newComplexQuery(ast *ComplexQuery, lhs queryFunc) queryFunc {
	rhs := newTerm(ast.Term)
	if ast.Operator == "OR" {
		return func(e LogEntry) bool { return lhs(e) || rhs(e) }
	}
	return func(e LogEntry) bool { return lhs(e) && rhs(e) }
}

func newCondition(ast *Condition) queryFunc {
	var fieldPrefix string
	if ast.FieldPrefix != nil {
		fieldPrefix = *ast.FieldPrefix
	}
	lhs := newFieldLoader(fieldPrefix, ast.FieldName)
	rhs := newExpr(ast.Value, false)

	if ast.Exclusion {
		return func(e LogEntry) bool { return !rhs(lhs(e)) }
	}
	return func(e LogEntry) bool { return rhs(lhs(e)) }
}

func newFreeCondition(ast *FreeCondition) queryFunc {
	lhs := newFieldLoader("", "message")
	rhs := newExpr(ast.Value, true)

	if ast.Exclusion {
		return func(e LogEntry) bool { return !rhs(lhs(e)) }
	}
	return func(e LogEntry) bool { return rhs(lhs(e)) }
}

func newExpr(ast *Expr, free bool) exprFunc {
	switch {
	case ast.Value != nil:
		return newValueExpr(ast.Value, free)
	case ast.SubExpr != nil:
		return newSubExpr(ast.SubExpr, free)
	default:
		panic("invalid expr")
	}
}

func newSubExpr(ast *SubExpr, free bool) exprFunc {
	var e exprFunc = newExpr(ast.First, free)
	for i := len(ast.Rest) - 1; i >= 0; i-- {
		e = newComplexExpr(ast.Rest[i], free, e)
	}
	return e
}

func newComplexExpr(ast *ComplexExpr, free bool, lhs exprFunc) exprFunc {
	rhs := newExpr(ast.Expr, free)
	switch ast.Operator {
	case "AND":
		return func(v any) bool { return lhs(v) && rhs(v) }
	case "OR":
		return func(v any) bool { return lhs(v) || rhs(v) }
	default:
		panic("invalid complex expr")
	}
}

func newValueExpr(ast *Value, free bool) exprFunc {
	switch {
	case ast.StringLit != nil:
		return newStringLitExpr(ast.StringLit.String(), free)
	case ast.String != nil:
		return newStringExpr(*ast.String, free)
	case ast.OpenRange != nil:
		return newOpenRangeExpr(ast.OpenRange)
	case ast.ClosedRange != nil:
		return newClosedRangeExpr(ast.ClosedRange)
	default:
		panic("invalid expr") // todo return error
	}
}

func newStringLitExpr(rhs string, free bool) exprFunc {
	if free {
		// freeform expressions are always substring matches. wildcards match
		// whitespace. this feels like some sketchy behavior maintained for
		// legacy compatibility. nothing we use relies on it and we shouldn't
		// use this but for the sake of completeness it's included here...
		return newStringExpr(strings.ReplaceAll(rhs, `*`, ` `), true)
	}
	return func(v any) bool {
		lhs, ok := v.(string)
		return ok && strings.EqualFold(lhs, rhs)
	}
}

func newStringExpr(rhs string, free bool) exprFunc {
	if free || strings.Contains(rhs, `*`) {
		pattern := strings.ReplaceAll(regexp.QuoteMeta(rhs), `\*`, `.*`)
		if free {
			// freeform expressions are case insensitive substring matches
			pattern = `(?i).*` + pattern + `.*`
		} else {
			// facet expressions are case sensitive full string matches
			pattern = `^` + pattern + `$`
		}
		rhs, err := regexp.Compile(pattern)
		if err != nil {
			panic(err)
		}

		return func(v any) bool {
			switch lhs := v.(type) {
			case string:
				return rhs.MatchString(lhs)
			case json.Number:
				return rhs.MatchString(string(lhs))
			default:
				return false
			}
		}
	}

	return func(v any) bool {
		switch lhs := v.(type) {
		case string:
			return lhs == rhs
		case json.Number:
			return string(lhs) == rhs
		default:
			return false
		}
	}
}

type openRangeOp int

const (
	openRangeGTE openRangeOp = iota
	openRangeLTE
	openRangeGE
	openRangeLE
)

func newOpenRangeExpr(ast *OpenRange) exprFunc {
	var op openRangeOp
	switch ast.Operator {
	case ">=":
		op = openRangeGTE
	case "<=":
		op = openRangeLTE
	case ">":
		op = openRangeGE
	case "<":
		op = openRangeLE
	}

	rhs := ast.Bound
	return func(v any) bool {
		lhs, ok := numberToFloat64(v)
		if !ok {
			return false
		}

		switch op {
		case openRangeGTE:
			return lhs >= rhs
		case openRangeLTE:
			return lhs <= rhs
		case openRangeGE:
			return lhs > rhs
		case openRangeLE:
			return lhs < rhs
		}
		return false
	}
}

func newClosedRangeExpr(ast *ClosedRange) exprFunc {
	min := ast.Min
	max := ast.Max
	return func(v any) bool {
		lhs, ok := numberToFloat64(v)
		return ok && min <= lhs && lhs <= max
	}
}
