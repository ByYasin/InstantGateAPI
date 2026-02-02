package query

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type FilterOperator string

const (
	OpEqual        FilterOperator = "eq"
	OpNotEqual     FilterOperator = "ne"
	OpGreater      FilterOperator = "gt"
	OpGreaterEqual FilterOperator = "gte"
	OpLess         FilterOperator = "lt"
	OpLessEqual    FilterOperator = "lte"
	OpLike         FilterOperator = "like"
	OpNotLike      FilterOperator = "nlike"
	OpIn           FilterOperator = "in"
	OpNotIn        FilterOperator = "nin"
)

type Filter struct {
	Field    string
	Operator FilterOperator
	Value    interface{}
	Values   []interface{}
}

type Pagination struct {
	Limit  int
	Offset int
	Page   int
}

type Sorting struct {
	Field     string
	Direction string
}

type QueryParams struct {
	Filters    []Filter
	Pagination *Pagination
	Sorting    *Sorting
	Fields     []string
}

func ParseFilters(r *http.Request) (*QueryParams, error) {
	params := &QueryParams{
		Filters:    make([]Filter, 0),
		Pagination: &Pagination{Limit: 50, Offset: 0},
	}

	query := r.URL.Query()

	for key, values := range query {
		if len(values) == 0 {
			continue
		}

		// Keep original case for column names, only lowercase for known params
		keyLower := strings.ToLower(key)

		switch keyLower {
		case "limit", "offset", "page", "order", "sort", "fields":
			continue
		default:
			// Use original key (with original case) for column names
			filter, err := parseFilter(key, values[0])
			if err != nil {
				return nil, fmt.Errorf("invalid filter '%s': %w", key, err)
			}
			if filter != nil {
				params.Filters = append(params.Filters, *filter)
			}
		}
	}

	params.Pagination = parsePagination(query)

	params.Sorting = parseSorting(query)

	params.Fields = parseFields(query)

	return params, nil
}

func parseFilter(field, value string) (*Filter, error) {
	if value == "" {
		return nil, nil
	}

	if strings.Contains(value, ".") {
		parts := strings.SplitN(value, ".", 2)
		if len(parts) == 2 {
			op := FilterOperator(strings.ToLower(parts[0]))
			val := parts[1]

			if isListOperator(op) {
				values := strings.Split(val, ",")
				return &Filter{
					Field:    field,
					Operator: op,
					Values:   parseValues(values),
				}, nil
			}

			parsedVal, err := parseValue(val)
			if err != nil {
				return nil, err
			}

			return &Filter{
				Field:    field,
				Operator: op,
				Value:    parsedVal,
			}, nil
		}
	}

	parsedVal, err := parseValue(value)
	if err != nil {
		return nil, err
	}

	return &Filter{
		Field:    field,
		Operator: OpEqual,
		Value:    parsedVal,
	}, nil
}

func isListOperator(op FilterOperator) bool {
	return op == OpIn || op == OpNotIn
}

func parseValue(value string) (interface{}, error) {
	value = strings.TrimSpace(value)

	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1], nil
		}
	}

	lower := strings.ToLower(value)
	if lower == "true" {
		return true, nil
	}
	if lower == "false" {
		return false, nil
	}
	if lower == "null" {
		return nil, nil
	}

	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i, nil
	}

	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f, nil
	}

	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	return value, nil
}

func parseValues(values []string) []interface{} {
	result := make([]interface{}, len(values))
	for i, v := range values {
		v = strings.TrimSpace(v)
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			result[i] = v[1 : len(v)-1]
			continue
		}
		result[i] = v
	}
	return result
}

func parsePagination(query url.Values) *Pagination {
	pag := &Pagination{
		Limit:  50,
		Offset: 0,
	}

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			pag.Limit = l
			if pag.Limit > 1000 {
				pag.Limit = 1000
			}
		}
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			pag.Offset = o
		}
	}

	if page := query.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			pag.Page = p
			pag.Offset = (p - 1) * pag.Limit
		}
	}

	return pag
}

func parseSorting(query url.Values) *Sorting {
	orderParam := query.Get("order")
	if orderParam == "" {
		orderParam = query.Get("sort")
	}

	if orderParam == "" {
		return nil
	}

	parts := strings.Split(orderParam, ".")
	sort := &Sorting{
		Field:     parts[0],
		Direction: "asc",
	}

	if len(parts) > 1 {
		direction := strings.ToLower(parts[1])
		if direction == "desc" || direction == "d" || direction == "-" {
			sort.Direction = "desc"
		}
	}

	return sort
}

func parseFields(query url.Values) []string {
	fieldsStr := query.Get("fields")
	if fieldsStr == "" {
		return nil
	}

	fields := strings.Split(fieldsStr, ",")
	result := make([]string, 0, len(fields))

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}

	return result
}

func IsValidOperator(op string) bool {
	switch FilterOperator(op) {
	case OpEqual, OpNotEqual, OpGreater, OpGreaterEqual,
		OpLess, OpLessEqual, OpLike, OpNotLike, OpIn, OpNotIn:
		return true
	default:
		return false
	}
}
