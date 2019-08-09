package sqlast

import (
	"testing"

	"github.com/andreyvit/diff"
)

func TestSQLSelect_ToSQLString(t *testing.T) {
	cases := []struct {
		name string
		in   *SQLSelect
		out  string
	}{
		{
			name: "simple select",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedSelectItem{
						Node: NewIdent("test"),
					},
				},
				FromClause: []TableReference{
					&Table{
						Name: NewSQLObjectName("test_table"),
					},
				},
			},
			out: "SELECT test FROM test_table",
		},
		{
			name: "join",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedSelectItem{
						Node: NewSQLObjectName("test"),
					},
				},
				FromClause: []TableReference{
					&NaturalJoin{
						LeftElement: &TableJoinElement{
							Ref: &Table{
								Name: NewSQLObjectName("test_table"),
							},
						},
						Type: IMPLICIT,
						RightElement: &TableJoinElement{
							Ref: &Table{
								Name: NewSQLObjectName("test_table2"),
							},
						},
					},
				},
			},
			out: "SELECT test FROM test_table NATURAL JOIN test_table2",
		},
		{
			name: "where",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedSelectItem{
						Node: NewIdent("test"),
					},
				},
				FromClause: []TableReference{
					&Table{
						Name: NewSQLObjectName("test_table"),
					},
				},
				WhereClause: &BinaryExpr{
					Left: &CompoundIdent{
						Idents: []*Ident{NewIdent("test_table"), NewIdent("column1")},
					},
					Op:    Eq,
					Right: NewSingleQuotedString("test"),
				},
			},
			out: "SELECT test FROM test_table WHERE test_table.column1 = 'test'",
		},
		{
			name: "count and join",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&AliasSelectItem{
						Expr: &Function{
							Name: NewSQLObjectName("COUNT"),
							Args: []Node{&CompoundIdent{
								Idents: []*Ident{NewIdent("t1"), NewIdent("id")},
							}},
						},
						Alias: NewIdent("c"),
					},
				},
				FromClause: []TableReference{
					&QualifiedJoin{
						LeftElement: &TableJoinElement{
							Ref: &Table{
								Name:  NewSQLObjectName("test_table"),
								Alias: NewIdent("t1"),
							},
						},
						Type: LEFT,
						RightElement: &TableJoinElement{
							Ref: &Table{
								Name:  NewSQLObjectName("test_table2"),
								Alias: NewIdent("t2"),
							},
						},
						Spec: &JoinCondition{
							SearchCondition: &BinaryExpr{
								Left: &CompoundIdent{
									Idents: []*Ident{NewIdent("t1"), NewIdent("id")},
								},
								Op: Eq,
								Right: &CompoundIdent{
									Idents: []*Ident{NewIdent("t2"), NewIdent("test_table_id")},
								},
							},
						},
					},
				},
			},
			out: "SELECT COUNT(t1.id) AS c FROM test_table AS t1 LEFT JOIN test_table2 AS t2 ON t1.id = t2.test_table_id",
		},
		{
			name: "group by",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedSelectItem{
						Node: &Function{
							Name: NewSQLObjectName("COUNT"),
							Args: []Node{NewIdent("customer_id")},
						},
					},
					&QualifiedWildcardSelectItem{
						Prefix: NewSQLObjectName("country"),
					},
				},
				FromClause: []TableReference{
					&Table{
						Name: NewSQLObjectName("customers"),
					},
				},
				GroupByClause: []Node{NewIdent("country")},
			},
			out: "SELECT COUNT(customer_id), country.* FROM customers GROUP BY country",
		},
		{
			name: "having",
			in: &SQLSelect{
				Projection: []SQLSelectItem{
					&UnnamedSelectItem{
						Node: &Function{
							Name: NewSQLObjectName("COUNT"),
							Args: []Node{NewIdent("customer_id")},
						},
					},
					&UnnamedSelectItem{
						Node: NewIdent("country"),
					},
				},
				FromClause: []TableReference{
					&Table{
						Name: NewSQLObjectName("customers"),
					},
				},
				GroupByClause: []Node{NewIdent("country")},
				HavingClause: &BinaryExpr{
					Op: Gt,
					Left: &Function{
						Name: NewSQLObjectName("COUNT"),
						Args: []Node{NewIdent("customer_id")},
					},
					Right: NewLongValue(3),
				},
			},
			out: "SELECT COUNT(customer_id), country FROM customers GROUP BY country HAVING COUNT(customer_id) > 3",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := c.in.ToSQLString()

			if act != c.out {
				t.Errorf("must be \n%s but \n%s \n diff: %s", c.out, act, diff.CharacterDiff(c.out, act))
			}
		})
	}

}

func TestSQLQuery_ToSQLString(t *testing.T) {
	cases := []struct {
		name string
		in   *Query
		out  string
	}{
		{
			// from https://www.postgresql.jp/document/9.3/html/queries-with.html
			name: "with cte",
			in: &Query{
				CTEs: []*CTE{
					{
						Alias: NewIdent("regional_sales"),
						Query: &Query{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedSelectItem{Node: NewIdent("region")},
									&AliasSelectItem{
										Alias: NewIdent("total_sales"),
										Expr: &Function{
											Name: NewSQLObjectName("SUM"),
											Args: []Node{NewIdent("amount")},
										},
									},
								},
								FromClause: []TableReference{
									&Table{
										Name: NewSQLObjectName("orders"),
									},
								},
								GroupByClause: []Node{NewIdent("region")},
							},
						},
					},
				},
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&UnnamedSelectItem{Node: NewIdent("product")},
						&AliasSelectItem{
							Alias: NewIdent("product_units"),
							Expr: &Function{
								Name: NewSQLObjectName("SUM"),
								Args: []Node{NewIdent("quantity")},
							},
						},
					},
					FromClause: []TableReference{
						&Table{
							Name: NewSQLObjectName("orders"),
						},
					},
					WhereClause: &InSubQuery{
						Expr: NewIdent("region"),
						SubQuery: &Query{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedSelectItem{Node: NewIdent("region")},
								},
								FromClause: []TableReference{
									&Table{
										Name: NewSQLObjectName("top_regions"),
									},
								},
							},
						},
					},
					GroupByClause: []Node{NewIdent("region"), NewIdent("product")},
				},
			},
			out: "WITH regional_sales AS (" +
				"SELECT region, SUM(amount) AS total_sales " +
				"FROM orders GROUP BY region) " +
				"SELECT product, SUM(quantity) AS product_units " +
				"FROM orders " +
				"WHERE region IN (SELECT region FROM top_regions) " +
				"GROUP BY region, product",
		},
		{
			name: "order by and limit",
			in: &Query{
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&UnnamedSelectItem{Node: NewIdent("product")},
						&AliasSelectItem{
							Alias: NewIdent("product_units"),
							Expr: &Function{
								Name: NewSQLObjectName("SUM"),
								Args: []Node{NewIdent("quantity")},
							},
						},
					},
					FromClause: []TableReference{
						&Table{
							Name: NewSQLObjectName("orders"),
						},
					},
					WhereClause: &InSubQuery{
						Expr: NewIdent("region"),
						SubQuery: &Query{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedSelectItem{Node: NewIdent("region")},
								},
								FromClause: []TableReference{
									&Table{
										Name: NewSQLObjectName("top_regions"),
									},
								},
							},
						},
					},
				},
				OrderBy: []*OrderByExpr{
					{Expr: NewIdent("product_units")},
				},
				Limit: &LimitExpr{LimitValue: NewLongValue(100)},
			},
			out: "SELECT product, SUM(quantity) AS product_units " +
				"FROM orders " +
				"WHERE region IN (SELECT region FROM top_regions) " +
				"ORDER BY product_units LIMIT 100",
		},
		{
			name: "exists",
			in: &Query{
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&UnnamedSelectItem{
							Node: &Wildcard{},
						},
					},
					FromClause: []TableReference{
						&Table{
							Name: NewSQLObjectName("user"),
						},
					},
					WhereClause: &Exists{
						Negated: true,
						Query: &Query{
							Body: &SQLSelect{
								Projection: []SQLSelectItem{
									&UnnamedSelectItem{
										Node: &Wildcard{},
									},
								},
								FromClause: []TableReference{
									&Table{
										Name: NewSQLObjectName("user_sub"),
									},
								},
								WhereClause: &BinaryExpr{
									Op: And,
									Left: &BinaryExpr{
										Op: Eq,
										Left: &CompoundIdent{
											Idents: []*Ident{
												NewIdent("user"),
												NewIdent("id"),
											},
										},
										Right: &CompoundIdent{
											Idents: []*Ident{
												NewIdent("user_sub"),
												NewIdent("id"),
											},
										},
									},
									Right: &BinaryExpr{
										Op: Eq,
										Left: &CompoundIdent{
											Idents: []*Ident{
												NewIdent("user_sub"),
												NewIdent("job"),
											},
										},
										Right: NewSingleQuotedString("job"),
									},
								},
							},
						},
					},
				},
			},
			out: "SELECT * FROM user WHERE NOT EXISTS (" +
				"SELECT * FROM user_sub WHERE user.id = user_sub.id AND user_sub.job = 'job'" +
				")",
		},
		{
			name: "between / case",
			in: &Query{
				Body: &SQLSelect{
					Projection: []SQLSelectItem{
						&AliasSelectItem{
							Expr: &CaseExpr{
								Conditions: []Node{
									&BinaryExpr{
										Op:    Eq,
										Left:  NewIdent("expr1"),
										Right: NewSingleQuotedString("1"),
									},
									&BinaryExpr{
										Op:    Eq,
										Left:  NewIdent("expr2"),
										Right: NewSingleQuotedString("2"),
									},
								},
								Results: []Node{
									NewSingleQuotedString("test1"),
									NewSingleQuotedString("test2"),
								},
								ElseResult: NewSingleQuotedString("other"),
							},
							Alias: NewIdent("alias"),
						},
					},
					FromClause: []TableReference{
						&Table{
							Name: NewSQLObjectName("user"),
						},
					},
					WhereClause: &Between{
						Expr: NewIdent("id"),
						High: NewLongValue(2),
						Low:  NewLongValue(1),
					},
				},
			},
			out: "SELECT CASE WHEN expr1 = '1' THEN 'test1' WHEN expr2 = '2' THEN 'test2' ELSE 'other' END AS alias " +
				"FROM user WHERE id BETWEEN 1 AND 2",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			act := c.in.ToSQLString()

			if act != c.out {
				t.Errorf("must be \n%s but \n%s \n diff: %s", c.out, act, diff.CharacterDiff(c.out, act))
			}
		})
	}

}
