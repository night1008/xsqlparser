package dialect

type ClickhouseDialect struct{}

func (*ClickhouseDialect) IsIdentifierStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '@' || r == '#'
}

func (*ClickhouseDialect) IsIdentifierPart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '@' || r == '_' || r == '#'
}

func (*ClickhouseDialect) IsDelimitedIdentifierStart(r rune) bool {
	return r == '"' || r == '`'
}

var _ Dialect = &ClickhouseDialect{}
