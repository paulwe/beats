package cloudmon

import (
	"bytes"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var lex = lexer.MustStateful(lexer.Rules{
	"Root": {
		{Name: "BooleanOperator", Pattern: `AND|OR`},
		{Name: "RangeOperator", Pattern: `<=|>=|<|>`},
		{Name: "FieldPrefix", Pattern: `@|#`},
		{Name: "SingleQuote", Pattern: `'`, Action: lexer.Push("SingleQuoteString")},
		{Name: "DoubleQuote", Pattern: `"`, Action: lexer.Push("DoubleQuoteString")},
		{Name: "Ident", Pattern: `[a-zA-Z0-9@#$%^&*+_\-='|;'<,>.?\x60~\\/]+`},
		{Name: "Punct", Pattern: `[-[!@#$%^&*()+_={}\|:;"'<,>.?\/]|]`},
		{Name: "EOL", Pattern: `[\n\r]+`},
		{Name: "Whitespace", Pattern: `[ \t]+`},
	},
	"SingleQuoteString": {
		{Name: "EscapedChar", Pattern: `\\.`},
		{Name: "SingleQuote", Pattern: `'`, Action: lexer.Pop()},
		{Name: "SingleQuoteStringChar", Pattern: `\$|[^$'\\]+`},
	},
	"DoubleQuoteString": {
		{Name: "EscapedChar", Pattern: `\\.`},
		{Name: "DoubleQuote", Pattern: `"`, Action: lexer.Pop()},
		{Name: "DoubleQuoteStringChar", Pattern: `\$|[^$"\\]+`},
	},
})

type ComplexExpr struct {
	Operator string `parser:"@BooleanOperator"`
	Expr     *Expr  `parser:"@@"`
}

type SubExpr struct {
	First *Expr          `parser:"@@"`
	Rest  []*ComplexExpr `parser:"@@*"`
}

type Expr struct {
	Value   *Value   `parser:"  @@"`
	SubExpr *SubExpr `parser:"| '(' @@ ')'"`
}

type Value struct {
	StringLit   StringFragments `parser:"  (SingleQuote @@* SingleQuote | DoubleQuote @@* DoubleQuote)"`
	String      *string         `parser:"| @Ident"`
	OpenRange   *OpenRange      `parser:"| @@"`
	ClosedRange *ClosedRange    `parser:"| '[' @@ ']'"`
}

type StringFragments []*StringFragment

func (s StringFragments) String() string {
	var b strings.Builder
	for _, f := range s {
		if f.Escaped != "" {
			b.WriteString(f.Escaped[1:])
		}
		b.WriteString(f.Chars)
	}
	return b.String()
}

type StringFragment struct {
	Escaped string `parser:"(  @EscapedChar"`
	Chars   string `parser:" | (@SingleQuoteStringChar | @DoubleQuoteStringChar))"`
}

type OpenRange struct {
	Operator string  `parser:"@RangeOperator"`
	Bound    float64 `parser:"@Ident"`
}

type ClosedRange struct {
	Min float64 `parser:"@Ident"`
	Max float64 `parser:"@( 'TO' Ident )"`
}

type Condition struct {
	Exclusion   bool    `parser:"@'-'?"`
	FieldPrefix *string `parser:"@FieldPrefix?"`
	FieldName   string  `parser:"(@Ident ':')"`
	Value       *Expr   `parser:"@@"`
}

type FreeCondition struct {
	Exclusion bool  `parser:"@'-'?"`
	Value     *Expr `parser:"@@"`
}

type Term struct {
	SubQuery      *SubQuery      `parser:"  @@"`
	Condition     *Condition     `parser:"| @@"`
	FreeCondition *FreeCondition `parser:"| @@"`
}

type SubQuery struct {
	Exclusion bool   `parser:"@'-'?"`
	Query     *Query `parser:"'(' @@ ')'"`
}

type ComplexQuery struct {
	Operator string `parser:"@BooleanOperator?"`
	Term     *Term  `parser:"@@"`
}

type Query struct {
	First *Term           `parser:"@@"`
	Rest  []*ComplexQuery `parser:"@@*"`
}

func NewQueryParser() *QueryParser {
	return &QueryParser{
		parser: participle.MustBuild[Query](
			participle.Lexer(lex),
			participle.Elide("Whitespace"),
			participle.CaseInsensitive("BooleanOperator"),
			participle.Upper("BooleanOperator"),
		),
	}
}

type QueryParser struct {
	parser *participle.Parser[Query]
}

func (p *QueryParser) Parse(query string) (*Query, error) {
	var buf bytes.Buffer
	q, err := p.parser.ParseString("", query, participle.Trace(&buf))
	return q, err
}
