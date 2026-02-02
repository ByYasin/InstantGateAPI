package query

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/proyaai/instantgate/internal/database/mysql"
)

type Builder struct {
	sb     sq.StatementBuilderType
	schema *mysql.SchemaCache
}

func NewBuilder(schema *mysql.SchemaCache) *Builder {
	return &Builder{
		sb:     sq.StatementBuilder.PlaceholderFormat(sq.Question),
		schema: schema,
	}
}

// escapeIdentifier wraps an identifier in backticks for MySQL
func escapeIdentifier(name string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(name, "`", "``"))
}

// escapeIdentifierSlice wraps all identifiers in backticks
func escapeIdentifierSlice(names []string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = escapeIdentifier(name)
	}
	return result
}

func (b *Builder) BuildSelect(table string, params *QueryParams) (string, []interface{}, error) {
	tableSchema, exists := b.schema.Get(table)
	if !exists {
		return "", nil, fmt.Errorf("table '%s' not found", table)
	}

	var columns []string
	if len(params.Fields) > 0 {
		for _, field := range params.Fields {
			if _, ok := tableSchema.Columns[strings.ToLower(field)]; !ok {
				return "", nil, fmt.Errorf("unknown column '%s' in table '%s'", field, table)
			}
			columns = append(columns, escapeIdentifier(field))
		}
	} else {
		// Use original column names from schema (not the lowercase keys)
		for _, col := range tableSchema.Columns {
			columns = append(columns, escapeIdentifier(col.Name))
		}
	}

	// Escape table name
	escapedTable := escapeIdentifier(table)
	query := b.sb.Select(columns...).From(escapedTable)

	for _, filter := range params.Filters {
		if _, ok := tableSchema.Columns[strings.ToLower(filter.Field)]; !ok {
			return "", nil, fmt.Errorf("unknown column '%s' in table '%s'", filter.Field, table)
		}

		query = applyFilter(query, filter)
	}

	if params.Sorting != nil {
		if _, ok := tableSchema.Columns[strings.ToLower(params.Sorting.Field)]; !ok {
			return "", nil, fmt.Errorf("unknown column '%s' for sorting", params.Sorting.Field)
		}
		orderClause := escapeIdentifier(params.Sorting.Field)
		if params.Sorting.Direction == "desc" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.OrderBy(orderClause)
	}

	if params.Pagination != nil {
		if params.Pagination.Limit > 0 {
			query = query.Limit(uint64(params.Pagination.Limit))
		}
		if params.Pagination.Offset > 0 {
			query = query.Offset(uint64(params.Pagination.Offset))
		}
	}

	return query.ToSql()
}

func (b *Builder) BuildSelectByID(table string, id interface{}, fields []string) (string, []interface{}, error) {
	tableSchema, exists := b.schema.Get(table)
	if !exists {
		return "", nil, fmt.Errorf("table '%s' not found", table)
	}

	if tableSchema.PrimaryKey == "" {
		return "", nil, fmt.Errorf("table '%s' has no primary key", table)
	}

	var columns []string
	if len(fields) > 0 {
		for _, field := range fields {
			if _, ok := tableSchema.Columns[strings.ToLower(field)]; !ok {
				return "", nil, fmt.Errorf("unknown column '%s' in table '%s'", field, table)
			}
			columns = append(columns, escapeIdentifier(field))
		}
	} else {
		// Use original column names from schema
		for _, col := range tableSchema.Columns {
			columns = append(columns, escapeIdentifier(col.Name))
		}
	}

	// Get the original PK column name (not lowercase key)
	pkCol, ok := tableSchema.Columns[tableSchema.PrimaryKey]
	if !ok {
		return "", nil, fmt.Errorf("primary key column not found")
	}

	escapedTable := escapeIdentifier(table)
	escapedPK := escapeIdentifier(pkCol.Name)

	query := b.sb.Select(columns...).
		From(escapedTable).
		Where(sq.Eq{escapedPK: id}).
		Limit(1)

	return query.ToSql()
}

func (b *Builder) BuildCount(table string, params *QueryParams) (string, []interface{}, error) {
	_, exists := b.schema.Get(table)
	if !exists {
		return "", nil, fmt.Errorf("table '%s' not found", table)
	}

	escapedTable := escapeIdentifier(table)
	query := b.sb.Select("COUNT(*) as count").From(escapedTable)

	for _, filter := range params.Filters {
		query = applyFilter(query, filter)
	}

	return query.ToSql()
}

func (b *Builder) BuildInsert(table string, data map[string]interface{}) (string, []interface{}, error) {
	tableSchema, exists := b.schema.Get(table)
	if !exists {
		return "", nil, fmt.Errorf("table '%s' not found", table)
	}

	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		colInfo, ok := tableSchema.Columns[strings.ToLower(col)]
		if !ok {
			return "", nil, fmt.Errorf("unknown column '%s' in table '%s'", col, table)
		}

		if colInfo.IsAutoIncrement {
			continue
		}

		// Use escaped column name from schema
		columns = append(columns, escapeIdentifier(colInfo.Name))
		values = append(values, val)
	}

	escapedTable := escapeIdentifier(table)
	query := b.sb.Insert(escapedTable).
		Columns(columns...).
		Values(values...)

	return query.ToSql()
}

func (b *Builder) BuildUpdate(table string, id interface{}, data map[string]interface{}) (string, []interface{}, error) {
	tableSchema, exists := b.schema.Get(table)
	if !exists {
		return "", nil, fmt.Errorf("table '%s' not found", table)
	}

	if tableSchema.PrimaryKey == "" {
		return "", nil, fmt.Errorf("table '%s' has no primary key", table)
	}

	updateData := make(map[string]interface{})
	for col, val := range data {
		colInfo, ok := tableSchema.Columns[strings.ToLower(col)]
		if !ok {
			return "", nil, fmt.Errorf("unknown column '%s' in table '%s'", col, table)
		}

		if colInfo.IsPrimaryKey || colInfo.IsAutoIncrement {
			continue
		}

		// Use escaped column name from schema
		updateData[escapeIdentifier(colInfo.Name)] = val
	}

	if len(updateData) == 0 {
		return "", nil, fmt.Errorf("no updateable columns provided")
	}

	// Get the original PK column name
	pkCol, ok := tableSchema.Columns[tableSchema.PrimaryKey]
	if !ok {
		return "", nil, fmt.Errorf("primary key column not found")
	}

	escapedTable := escapeIdentifier(table)
	escapedPK := escapeIdentifier(pkCol.Name)

	query := b.sb.Update(escapedTable).
		SetMap(updateData).
		Where(sq.Eq{escapedPK: id})

	return query.ToSql()
}

func (b *Builder) BuildDelete(table string, id interface{}) (string, []interface{}, error) {
	tableSchema, exists := b.schema.Get(table)
	if !exists {
		return "", nil, fmt.Errorf("table '%s' not found", table)
	}

	if tableSchema.PrimaryKey == "" {
		return "", nil, fmt.Errorf("table '%s' has no primary key", table)
	}

	// Get the original PK column name
	pkCol, ok := tableSchema.Columns[tableSchema.PrimaryKey]
	if !ok {
		return "", nil, fmt.Errorf("primary key column not found")
	}

	escapedTable := escapeIdentifier(table)
	escapedPK := escapeIdentifier(pkCol.Name)

	query := b.sb.Delete(escapedTable).
		Where(sq.Eq{escapedPK: id})

	return query.ToSql()
}

func applyFilter(query sq.SelectBuilder, filter Filter) sq.SelectBuilder {
	// Escape column name for filters
	escapedField := escapeIdentifier(filter.Field)

	switch filter.Operator {
	case OpEqual:
		return query.Where(sq.Eq{escapedField: filter.Value})
	case OpNotEqual:
		return query.Where(sq.NotEq{escapedField: filter.Value})
	case OpGreater:
		return query.Where(sq.Gt{escapedField: filter.Value})
	case OpGreaterEqual:
		return query.Where(sq.GtOrEq{escapedField: filter.Value})
	case OpLess:
		return query.Where(sq.Lt{escapedField: filter.Value})
	case OpLessEqual:
		return query.Where(sq.LtOrEq{escapedField: filter.Value})
	case OpLike:
		return query.Where(sq.Like{escapedField: filter.Value})
	case OpNotLike:
		return query.Where(sq.NotLike{escapedField: filter.Value})
	case OpIn:
		return query.Where(sq.Eq{escapedField: filter.Values})
	case OpNotIn:
		return query.Where(sq.NotEq{escapedField: filter.Values})
	default:
		return query.Where(sq.Eq{escapedField: filter.Value})
	}
}
