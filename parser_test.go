package xsqlparser

import (
	"bytes"
	"reflect"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"

	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
)

var IgnoreMarker = cmp.FilterPath(func(paths cmp.Path) bool {
	s := paths.Last().Type()
	name := s.Name()
	r := []rune(name)
	return s.Kind() == reflect.Struct && len(r) > 0 && unicode.IsLower(r[0])
}, cmp.Ignore())

func TestParser_ParseStatement(t *testing.T) {
	t.Run("select", func(t *testing.T) {

		cases := []struct {
			name string
			in   string
			out  sqlast.SQLStmt
			skip bool
		}{
			{
				name: "simple select",
				in:   "SELECT test FROM test_table",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{
								Node: sqlast.NewIdent("test"),
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("test_table"),
							},
						},
					},
				},
			},
			{
				name: "where",
				in:   "SELECT test FROM test_table WHERE test_table.column1 = 'test'",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{
								Node: sqlast.NewIdent("test"),
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("test_table"),
							},
						},
						WhereClause: &sqlast.SQLBinaryExpr{
							Left: &sqlast.SQLCompoundIdentifier{
								Idents: []*sqlast.Ident{sqlast.NewIdent("test_table"), sqlast.NewIdent("column1")},
							},
							Op:    sqlast.Eq,
							Right: sqlast.NewSingleQuotedString("test"),
						},
					},
				},
			},
			{
				name: "count and join",
				in:   "SELECT COUNT(t1.id) AS c FROM test_table AS t1 LEFT JOIN test_table2 AS t2 ON t1.id = t2.test_table_id",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.ExpressionWithAlias{
								Expr: &sqlast.SQLFunction{
									Name: sqlast.NewSQLObjectName("COUNT"),
									Args: []sqlast.Node{&sqlast.SQLCompoundIdentifier{
										Idents: []*sqlast.Ident{sqlast.NewIdent("t1"), sqlast.NewIdent("id")},
									}},
								},
								Alias: sqlast.NewIdent("c"),
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.QualifiedJoin{
								LeftElement: &sqlast.TableJoinElement{
									Ref: &sqlast.Table{
										Name:  sqlast.NewSQLObjectName("test_table"),
										Alias: sqlast.NewIdent("t1"),
									},
								},
								Type: sqlast.LEFT,
								RightElement: &sqlast.TableJoinElement{
									Ref: &sqlast.Table{
										Name:  sqlast.NewSQLObjectName("test_table2"),
										Alias: sqlast.NewIdent("t2"),
									},
								},
								Spec: &sqlast.JoinCondition{
									SearchCondition: &sqlast.SQLBinaryExpr{
										Left: &sqlast.SQLCompoundIdentifier{
											Idents: []*sqlast.Ident{sqlast.NewIdent("t1"), sqlast.NewIdent("id")},
										},
										Op: sqlast.Eq,
										Right: &sqlast.SQLCompoundIdentifier{
											Idents: []*sqlast.Ident{sqlast.NewIdent("t2"), sqlast.NewIdent("test_table_id")},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				name: "group by",
				in:   "SELECT COUNT(customer_id), country.* FROM customers GROUP BY country",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{
								Node: &sqlast.SQLFunction{
									Name: sqlast.NewSQLObjectName("COUNT"),
									Args: []sqlast.Node{sqlast.NewIdent("customer_id")},
								},
							},
							&sqlast.QualifiedWildcard{
								Prefix: sqlast.NewSQLObjectName("country"),
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("customers"),
							},
						},
						GroupByClause: []sqlast.Node{sqlast.NewIdent("country")},
					},
				},
			},
			{
				name: "having",
				in:   "SELECT COUNT(customer_id), country FROM customers GROUP BY country HAVING COUNT(customer_id) > 3",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{
								Node: &sqlast.SQLFunction{
									Name: sqlast.NewSQLObjectName("COUNT"),
									Args: []sqlast.Node{sqlast.NewIdent("customer_id")},
								},
							},
							&sqlast.UnnamedExpression{
								Node: sqlast.NewIdent("country"),
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("customers"),
							},
						},
						GroupByClause: []sqlast.Node{sqlast.NewIdent("country")},
						HavingClause: &sqlast.SQLBinaryExpr{
							Op: sqlast.Gt,
							Left: &sqlast.SQLFunction{
								Name: sqlast.NewSQLObjectName("COUNT"),
								Args: []sqlast.Node{sqlast.NewIdent("customer_id")},
							},
							Right: sqlast.NewLongValue(3),
						},
					},
				},
			},
			{
				name: "order by and limit",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{Node: sqlast.NewIdent("product")},
							&sqlast.ExpressionWithAlias{
								Alias: sqlast.NewIdent("product_units"),
								Expr: &sqlast.SQLFunction{
									Name: sqlast.NewSQLObjectName("SUM"),
									Args: []sqlast.Node{sqlast.NewIdent("quantity")},
								},
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("orders"),
							},
						},
						WhereClause: &sqlast.SQLInSubQuery{
							Expr: sqlast.NewIdent("region"),
							SubQuery: &sqlast.SQLQuery{
								Body: &sqlast.SQLSelect{
									Projection: []sqlast.SQLSelectItem{
										&sqlast.UnnamedExpression{Node: sqlast.NewIdent("region")},
									},
									FromClause: []sqlast.TableReference{
										&sqlast.Table{
											Name: sqlast.NewSQLObjectName("top_regions"),
										},
									},
								},
							},
						},
					},
					OrderBy: []*sqlast.SQLOrderByExpr{
						{Expr: sqlast.NewIdent("product_units")},
					},
					Limit: &sqlast.LimitExpr{
						LimitValue: sqlast.NewLongValue(100),
					},
				},
				in: "SELECT product, SUM(quantity) AS product_units " +
					"FROM orders " +
					"WHERE region IN (SELECT region FROM top_regions) " +
					"ORDER BY product_units LIMIT 100",
			},
			{
				// from https://www.postgresql.jp/document/9.3/html/queries-with.html
				name: "with cte",
				out: &sqlast.SQLQuery{
					CTEs: []*sqlast.CTE{
						{
							Alias: sqlast.NewIdent("regional_sales"),
							Query: &sqlast.SQLQuery{
								Body: &sqlast.SQLSelect{
									Projection: []sqlast.SQLSelectItem{
										&sqlast.UnnamedExpression{Node: sqlast.NewIdent("region")},
										&sqlast.ExpressionWithAlias{
											Alias: sqlast.NewIdent("total_sales"),
											Expr: &sqlast.SQLFunction{
												Name: sqlast.NewSQLObjectName("SUM"),
												Args: []sqlast.Node{sqlast.NewIdent("amount")},
											},
										},
									},
									FromClause: []sqlast.TableReference{
										&sqlast.Table{
											Name: sqlast.NewSQLObjectName("orders"),
										},
									},
									GroupByClause: []sqlast.Node{sqlast.NewIdent("region")},
								},
							},
						},
					},
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{Node: sqlast.NewIdent("product")},
							&sqlast.ExpressionWithAlias{
								Alias: sqlast.NewIdent("product_units"),
								Expr: &sqlast.SQLFunction{
									Name: sqlast.NewSQLObjectName("SUM"),
									Args: []sqlast.Node{sqlast.NewIdent("quantity")},
								},
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("orders"),
							},
						},
						WhereClause: &sqlast.SQLInSubQuery{
							Expr: sqlast.NewIdent("region"),
							SubQuery: &sqlast.SQLQuery{
								Body: &sqlast.SQLSelect{
									Projection: []sqlast.SQLSelectItem{
										&sqlast.UnnamedExpression{Node: sqlast.NewIdent("region")},
									},
									FromClause: []sqlast.TableReference{
										&sqlast.Table{
											Name: sqlast.NewSQLObjectName("top_regions"),
										},
									},
								},
							},
						},
						GroupByClause: []sqlast.Node{sqlast.NewIdent("region"), sqlast.NewIdent("product")},
					},
				},
				in: "WITH regional_sales AS (" +
					"SELECT region, SUM(amount) AS total_sales " +
					"FROM orders GROUP BY region) " +
					"SELECT product, SUM(quantity) AS product_units " +
					"FROM orders " +
					"WHERE region IN (SELECT region FROM top_regions) " +
					"GROUP BY region, product",
			},
			{
				name: "exists",
				in: "SELECT * FROM user WHERE NOT EXISTS (" +
					"SELECT * FROM user_sub WHERE user.id = user_sub.id AND user_sub.job = 'job'" +
					");",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.UnnamedExpression{
								Node: &sqlast.SQLWildcard{},
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("user"),
							},
						},
						WhereClause: &sqlast.SQLExists{
							Negated: true,
							Query: &sqlast.SQLQuery{
								Body: &sqlast.SQLSelect{
									Projection: []sqlast.SQLSelectItem{
										&sqlast.UnnamedExpression{
											Node: &sqlast.SQLWildcard{},
										},
									},
									FromClause: []sqlast.TableReference{
										&sqlast.Table{
											Name: sqlast.NewSQLObjectName("user_sub"),
										},
									},
									WhereClause: &sqlast.SQLBinaryExpr{
										Op: sqlast.And,
										Left: &sqlast.SQLBinaryExpr{
											Op: sqlast.Eq,
											Left: &sqlast.SQLCompoundIdentifier{
												Idents: []*sqlast.Ident{
													sqlast.NewIdent("user"),
													sqlast.NewIdent("id"),
												},
											},
											Right: &sqlast.SQLCompoundIdentifier{
												Idents: []*sqlast.Ident{
													sqlast.NewIdent("user_sub"),
													sqlast.NewIdent("id"),
												},
											},
										},
										Right: &sqlast.SQLBinaryExpr{
											Op: sqlast.Eq,
											Left: &sqlast.SQLCompoundIdentifier{
												Idents: []*sqlast.Ident{
													sqlast.NewIdent("user_sub"),
													sqlast.NewIdent("job"),
												},
											},
											Right: sqlast.NewSingleQuotedString("job"),
										},
									},
								},
							},
						},
					},
				},
			},
			{
				name: "between / case",
				in: "SELECT CASE WHEN expr1 = '1' THEN 'test1' WHEN expr2 = '2' THEN 'test2' ELSE 'other' END AS alias " +
					"FROM user WHERE id BETWEEN 1 AND 2",
				out: &sqlast.SQLQuery{
					Body: &sqlast.SQLSelect{
						Projection: []sqlast.SQLSelectItem{
							&sqlast.ExpressionWithAlias{
								Expr: &sqlast.SQLCase{
									Conditions: []sqlast.Node{
										&sqlast.SQLBinaryExpr{
											Op:    sqlast.Eq,
											Left:  sqlast.NewIdent("expr1"),
											Right: sqlast.NewSingleQuotedString("1"),
										},
										&sqlast.SQLBinaryExpr{
											Op:    sqlast.Eq,
											Left:  sqlast.NewIdent("expr2"),
											Right: sqlast.NewSingleQuotedString("2"),
										},
									},
									Results: []sqlast.Node{
										sqlast.NewSingleQuotedString("test1"),
										sqlast.NewSingleQuotedString("test2"),
									},
									ElseResult: sqlast.NewSingleQuotedString("other"),
								},
								Alias: sqlast.NewIdent("alias"),
							},
						},
						FromClause: []sqlast.TableReference{
							&sqlast.Table{
								Name: sqlast.NewSQLObjectName("user"),
							},
						},
						WhereClause: &sqlast.SQLBetween{
							Expr: sqlast.NewIdent("id"),
							High: sqlast.NewLongValue(2),
							Low:  sqlast.NewLongValue(1),
						},
					},
				},
			},
		}

		for _, c := range cases {

			t.Run(c.name, func(t *testing.T) {
				if c.skip {
					t.Skip()
				}
				parser, err := NewParser(bytes.NewBufferString(c.in), &dialect.GenericSQLDialect{})
				if err != nil {
					t.Fatal(err)
				}
				ast, err := parser.ParseStatement()
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(c.out, ast, IgnoreMarker); diff != "" {
					t.Errorf("diff %s", diff)
				}
			})
		}
	})

	t.Run("create", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			out  sqlast.SQLStmt
			skip bool
		}{
			{
				name: "create table",
				in: "CREATE TABLE persons (" +
					"person_id UUID PRIMARY KEY NOT NULL, " +
					"first_name varchar(255) UNIQUE, " +
					"last_name character varying(255) NOT NULL, " +
					"created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				out: &sqlast.SQLCreateTable{
					Name: sqlast.NewSQLObjectName("persons"),
					Elements: []sqlast.TableElement{
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("person_id"),
							DataType: &sqlast.UUID{},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.UniqueColumnSpec{
										IsPrimaryKey: true,
									},
								},
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name: sqlast.NewIdent("first_name"),
							DataType: &sqlast.VarcharType{
								Size: sqlast.NewSize(255),
							},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.UniqueColumnSpec{},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name: sqlast.NewIdent("last_name"),
							DataType: &sqlast.VarcharType{
								Size: sqlast.NewSize(255),
							},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("created_at"),
							DataType: &sqlast.Timestamp{},
							Default:  sqlast.NewIdent("CURRENT_TIMESTAMP"),
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
					},
				},
			},
			{
				name: "with case",
				in: "CREATE TABLE persons (" +
					"person_id int PRIMARY KEY NOT NULL, " +
					"last_name character varying(255) NOT NULL, " +
					"test_id int NOT NULL REFERENCES test(id1, id2), " +
					"email character varying(255) UNIQUE NOT NULL, " +
					"age int NOT NULL CHECK(age > 0 AND age < 100), " +
					"created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL)",
				out: &sqlast.SQLCreateTable{
					Name: sqlast.NewSQLObjectName("persons"),
					Elements: []sqlast.TableElement{
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("person_id"),
							DataType: &sqlast.Int{},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.UniqueColumnSpec{
										IsPrimaryKey: true,
									},
								},
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name: sqlast.NewIdent("last_name"),
							DataType: &sqlast.VarcharType{
								Size: sqlast.NewSize(255),
							},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("test_id"),
							DataType: &sqlast.Int{},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
								{
									Spec: &sqlast.ReferencesColumnSpec{
										TableName: sqlast.NewSQLObjectName("test"),
										Columns:   []*sqlast.Ident{sqlast.NewIdent("id1"), sqlast.NewIdent("id2")},
									},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name: sqlast.NewIdent("email"),
							DataType: &sqlast.VarcharType{
								Size: sqlast.NewSize(255),
							},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.UniqueColumnSpec{},
								},
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("age"),
							DataType: &sqlast.Int{},
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
								{
									Spec: &sqlast.CheckColumnSpec{
										Expr: &sqlast.SQLBinaryExpr{
											Op: sqlast.And,
											Left: &sqlast.SQLBinaryExpr{
												Op:    sqlast.Gt,
												Left:  sqlast.NewIdent("age"),
												Right: sqlast.NewLongValue(0),
											},
											Right: &sqlast.SQLBinaryExpr{
												Op:    sqlast.Lt,
												Left:  sqlast.NewIdent("age"),
												Right: sqlast.NewLongValue(100),
											},
										},
									},
								},
							},
						},
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("created_at"),
							DataType: &sqlast.Timestamp{},
							Default:  sqlast.NewIdent("CURRENT_TIMESTAMP"),
							Constraints: []*sqlast.ColumnConstraint{
								{
									Spec: &sqlast.NotNullColumnSpec{},
								},
							},
						},
					},
				},
			},
			{
				name: "with table constraint",
				in: "CREATE TABLE persons (" +
					"person_id int, " +
					"CONSTRAINT production UNIQUE(test_column), " +
					"PRIMARY KEY(person_id), " +
					"CHECK(id > 100), " +
					"FOREIGN KEY(test_id) REFERENCES other_table(col1, col2)" +
					")",
				out: &sqlast.SQLCreateTable{
					Name: sqlast.NewSQLObjectName("persons"),
					Elements: []sqlast.TableElement{
						&sqlast.SQLColumnDef{
							Name:     sqlast.NewIdent("person_id"),
							DataType: &sqlast.Int{},
						},
						&sqlast.TableConstraint{
							Name: sqlast.NewIdent("production"),
							Spec: &sqlast.UniqueTableConstraint{
								Columns: []*sqlast.Ident{sqlast.NewIdent("test_column")},
							},
						},
						&sqlast.TableConstraint{
							Spec: &sqlast.UniqueTableConstraint{
								Columns:   []*sqlast.Ident{sqlast.NewIdent("person_id")},
								IsPrimary: true,
							},
						},
						&sqlast.TableConstraint{
							Spec: &sqlast.CheckTableConstraint{
								Expr: &sqlast.SQLBinaryExpr{
									Left:  sqlast.NewIdent("id"),
									Op:    sqlast.Gt,
									Right: sqlast.NewLongValue(100),
								},
							},
						},
						&sqlast.TableConstraint{
							Spec: &sqlast.ReferentialTableConstraint{
								Columns: []*sqlast.Ident{sqlast.NewIdent("test_id")},
								KeyExpr: &sqlast.ReferenceKeyExpr{
									TableName: sqlast.NewIdent("other_table"),
									Columns:   []*sqlast.Ident{sqlast.NewIdent("col1"), sqlast.NewIdent("col2")},
								},
							},
						},
					},
				},
			},
			{
				name: "create view",
				in:   "CREATE VIEW comedies AS SELECT * FROM films WHERE kind = 'Comedy'",
				out: &sqlast.SQLCreateView{
					Name: sqlast.NewSQLObjectName("comedies"),
					Query: &sqlast.SQLQuery{
						Body: &sqlast.SQLSelect{
							Projection: []sqlast.SQLSelectItem{&sqlast.UnnamedExpression{Node: &sqlast.SQLWildcard{}}},
							FromClause: []sqlast.TableReference{
								&sqlast.Table{
									Name: sqlast.NewSQLObjectName("films"),
								},
							},
							WhereClause: &sqlast.SQLBinaryExpr{
								Op:    sqlast.Eq,
								Left:  sqlast.NewIdent("kind"),
								Right: sqlast.NewSingleQuotedString("Comedy"),
							},
						},
					},
				},
			},
		}

		for _, c := range cases {

			t.Run(c.name, func(t *testing.T) {
				if c.skip {
					t.Skip()
				}
				parser, err := NewParser(bytes.NewBufferString(c.in), &dialect.GenericSQLDialect{})
				if err != nil {
					t.Fatal(err)
				}
				ast, err := parser.ParseStatement()
				if err != nil {
					t.Fatalf("%+v", err)
				}

				if diff := cmp.Diff(c.out, ast, IgnoreMarker); diff != "" {
					t.Errorf("diff %s", diff)
				}
			})
		}
	})

	t.Run("delete", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			out  sqlast.SQLStmt
			skip bool
		}{
			{
				in:   "DELETE FROM customers WHERE customer_id = 1",
				name: "simple case",
				out: &sqlast.SQLDelete{
					TableName: sqlast.NewSQLObjectName("customers"),
					Selection: &sqlast.SQLBinaryExpr{
						Op:    sqlast.Eq,
						Left:  sqlast.NewIdent("customer_id"),
						Right: sqlast.NewLongValue(1),
					},
				},
			},
		}

		for _, c := range cases {

			t.Run(c.name, func(t *testing.T) {
				if c.skip {
					t.Skip()
				}
				parser, err := NewParser(bytes.NewBufferString(c.in), &dialect.GenericSQLDialect{})
				if err != nil {
					t.Fatal(err)
				}
				ast, err := parser.ParseStatement()
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(c.out, ast, IgnoreMarker); diff != "" {
					t.Errorf("diff %s", diff)
				}
			})
		}
	})

	t.Run("insert", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			out  sqlast.SQLStmt
			skip bool
		}{
			{
				in:   "INSERT INTO customers (customer_name, contract_name) VALUES('Cardinal', 'Tom B. Erichsen')",
				name: "simple case",
				out: &sqlast.SQLInsert{
					TableName: sqlast.NewSQLObjectName("customers"),
					Columns: []*sqlast.Ident{
						sqlast.NewIdent("customer_name"),
						sqlast.NewIdent("contract_name"),
					},
					Values: [][]sqlast.Node{
						{
							sqlast.NewSingleQuotedString("Cardinal"),
							sqlast.NewSingleQuotedString("Tom B. Erichsen"),
						},
					},
				},
			},
			{
				name: "multi record case",
				in: "INSERT INTO customers (customer_name, contract_name) VALUES" +
					"('Cardinal', 'Tom B. Erichsen')," +
					"('Cardinal', 'Tom B. Erichsen')",
				out: &sqlast.SQLInsert{
					TableName: sqlast.NewSQLObjectName("customers"),
					Columns: []*sqlast.Ident{
						sqlast.NewIdent("customer_name"),
						sqlast.NewIdent("contract_name"),
					},
					Values: [][]sqlast.Node{
						{
							sqlast.NewSingleQuotedString("Cardinal"),
							sqlast.NewSingleQuotedString("Tom B. Erichsen"),
						},
						{
							sqlast.NewSingleQuotedString("Cardinal"),
							sqlast.NewSingleQuotedString("Tom B. Erichsen"),
						},
					},
				},
			},
		}

		for _, c := range cases {

			t.Run(c.name, func(t *testing.T) {
				if c.skip {
					t.Skip()
				}
				parser, err := NewParser(bytes.NewBufferString(c.in), &dialect.GenericSQLDialect{})
				if err != nil {
					t.Fatal(err)
				}
				ast, err := parser.ParseStatement()
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(c.out, ast, IgnoreMarker); diff != "" {
					t.Errorf("diff %s", diff)
				}
			})
		}
	})

	t.Run("alter", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			out  sqlast.SQLStmt
			skip bool
		}{
			{
				name: "add column",
				out: &sqlast.SQLAlterTable{
					TableName: sqlast.NewSQLObjectName("customers"),
					Action: &sqlast.AddColumnTableAction{
						Column: &sqlast.SQLColumnDef{
							Name: sqlast.NewIdent("email"),
							DataType: &sqlast.VarcharType{
								Size: sqlast.NewSize(255),
							},
						},
					},
				},
				in: "ALTER TABLE customers " +
					"ADD COLUMN email character varying(255)",
			},
			{
				name: "add constraint",
				out: &sqlast.SQLAlterTable{
					TableName: sqlast.NewSQLObjectName("products"),
					Action: &sqlast.AddConstraintTableAction{
						Constraint: &sqlast.TableConstraint{
							Spec: &sqlast.ReferentialTableConstraint{
								Columns: []*sqlast.Ident{sqlast.NewIdent("test_id")},
								KeyExpr: &sqlast.ReferenceKeyExpr{
									TableName: sqlast.NewIdent("other_table"),
									Columns:   []*sqlast.Ident{sqlast.NewIdent("col1"), sqlast.NewIdent("col2")},
								},
							},
						},
					},
				},
				in: "ALTER TABLE products " +
					"ADD FOREIGN KEY(test_id) REFERENCES other_table(col1, col2)",
			},
			{
				name: "drop constraint",
				out: &sqlast.SQLAlterTable{
					TableName: sqlast.NewSQLObjectName("products"),
					Action: &sqlast.DropConstraintTableAction{
						Name:    sqlast.NewIdent("fk"),
						Cascade: true,
					},
				},
				in: "ALTER TABLE products " +
					"DROP CONSTRAINT fk CASCADE",
			},
			{
				name: "remove column",
				out: &sqlast.SQLAlterTable{
					TableName: sqlast.NewSQLObjectName("products"),
					Action: &sqlast.RemoveColumnTableAction{
						Name:    sqlast.NewIdent("description"),
						Cascade: true,
					},
				},
				in: "ALTER TABLE products " +
					"DROP COLUMN description CASCADE",
			},
			{
				name: "alter column",
				out: &sqlast.SQLAlterTable{
					TableName: sqlast.NewSQLObjectName("products"),
					Action: &sqlast.AlterColumnTableAction{
						ColumnName: sqlast.NewIdent("created_at"),
						Action: &sqlast.SetDefaultColumnAction{
							Default: sqlast.NewIdent("current_timestamp"),
						},
					},
				},
				in: "ALTER TABLE products " +
					"ALTER COLUMN created_at SET DEFAULT current_timestamp",
			},
			{
				name: "pg change type",
				out: &sqlast.SQLAlterTable{
					TableName: sqlast.NewSQLObjectName("products"),
					Action: &sqlast.AlterColumnTableAction{
						ColumnName: sqlast.NewIdent("number"),
						Action: &sqlast.PGAlterDataTypeColumnAction{
							DataType: &sqlast.Decimal{
								Scale:     sqlast.NewSize(10),
								Precision: sqlast.NewSize(255),
							},
						},
					},
				},
				in: "ALTER TABLE products " +
					"ALTER COLUMN number TYPE numeric(255,10)",
			},
		}

		for _, c := range cases {

			t.Run(c.name, func(t *testing.T) {
				if c.skip {
					t.Skip()
				}
				parser, err := NewParser(bytes.NewBufferString(c.in), &dialect.GenericSQLDialect{})
				if err != nil {
					t.Fatal(err)
				}
				ast, err := parser.ParseStatement()
				if err != nil {
					t.Fatalf("%+v", err)
				}

				if diff := cmp.Diff(c.out, ast, IgnoreMarker); diff != "" {
					t.Errorf("diff %s", diff)
				}
			})
		}
	})

	t.Run("update", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			out  sqlast.SQLStmt
			skip bool
		}{
			{
				name: "simple case",
				in:   "UPDATE customers SET contract_name = 'Alfred Schmidt', city = 'Frankfurt' WHERE customer_id = 1",
				out: &sqlast.SQLUpdate{
					TableName: sqlast.NewSQLObjectName("customers"),
					Assignments: []*sqlast.SQLAssignment{
						{
							ID:    sqlast.NewIdent("contract_name"),
							Value: sqlast.NewSingleQuotedString("Alfred Schmidt"),
						},
						{
							ID:    sqlast.NewIdent("city"),
							Value: sqlast.NewSingleQuotedString("Frankfurt"),
						},
					},
					Selection: &sqlast.SQLBinaryExpr{
						Op:    sqlast.Eq,
						Left:  sqlast.NewIdent("customer_id"),
						Right: sqlast.NewLongValue(1),
					},
				},
			},
		}

		for _, c := range cases {

			t.Run(c.name, func(t *testing.T) {
				if c.skip {
					t.Skip()
				}
				parser, err := NewParser(bytes.NewBufferString(c.in), &dialect.GenericSQLDialect{})
				if err != nil {
					t.Fatal(err)
				}
				ast, err := parser.ParseStatement()
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(c.out, ast, IgnoreMarker); diff != "" {
					t.Errorf("diff %s", diff)
				}
			})
		}
	})
}

func TestParser_ParseSQL(t *testing.T) {
	in := `
create table account (
    account_id serial primary key,
    name varchar(255) not null,
    email varchar(255) unique not null,
    age smallint not null,
    registered_at timestamp with time zone default current_timestamp
);

create table category (
    category_id serial primary key,
    name varchar(255) not null
);

create table item (
    item_id serial primary key,
    price int not null,
    name varchar(255) not null,
    category_id int references category(category_id),
    created_at timestamp with time zone default current_timestamp
);
`
	parser, err := NewParser(bytes.NewBufferString(in), &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	stmts, err := parser.ParseSQL()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(stmts) != 3 {
		t.Fatal("must be 3 stmts")
	}
}
