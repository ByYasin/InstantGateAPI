package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/proyaai/instantgate/internal/database/mysql"
	"github.com/proyaai/instantgate/internal/query"
)

type GenericHandler struct {
	db      *sql.DB
	schema  *mysql.SchemaCache
	builder *query.Builder
}

func NewGenericHandler(db *sql.DB, schema *mysql.SchemaCache) *GenericHandler {
	return &GenericHandler{
		db:      db,
		schema:  schema,
		builder: query.NewBuilder(schema),
	}
}

func (h *GenericHandler) ListTable(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")

	if !h.schema.TableExists(tableName) {
		SendError(w, r, http.StatusNotFound, ErrTableNotFound, nil)
		return
	}

	params, err := query.ParseFilters(r)
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidFilter, err)
		return
	}

	selectSQL, args, err := h.builder.BuildSelect(tableName, params)
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, err)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), selectSQL, args...)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}

	countSQL, countArgs, _ := h.builder.BuildCount(tableName, params)
	var totalCount int64
	if countSQL != "" {
		h.db.QueryRowContext(r.Context(), countSQL, countArgs...).Scan(&totalCount)
	}

	response := map[string]interface{}{
		"data":  results,
		"count": len(results),
		"pagination": map[string]interface{}{
			"limit":  params.Pagination.Limit,
			"offset": params.Pagination.Offset,
			"total":  totalCount,
		},
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
	w.Header().Set("X-Limit", strconv.Itoa(params.Pagination.Limit))
	w.Header().Set("X-Offset", strconv.Itoa(params.Pagination.Offset))

	SendJSON(w, r, http.StatusOK, response)
}

func (h *GenericHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	id := chi.URLParam(r, "id")

	tableSchema, exists := h.schema.Get(tableName)
	if !exists {
		SendError(w, r, http.StatusNotFound, ErrTableNotFound, nil)
		return
	}

	if tableSchema.PrimaryKey == "" {
		SendError(w, r, http.StatusBadRequest, "Table has no primary key", nil)
		return
	}

	params, _ := query.ParseFilters(r)

	selectSQL, args, err := h.builder.BuildSelectByID(tableName, id, params.Fields)
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, err)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), selectSQL, args...)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}

	if len(results) == 0 {
		SendError(w, r, http.StatusNotFound, ErrRecordNotFound, nil)
		return
	}

	SendJSON(w, r, http.StatusOK, results[0])
}

func (h *GenericHandler) Create(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")

	if !h.schema.TableExists(tableName) {
		SendError(w, r, http.StatusNotFound, ErrTableNotFound, nil)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidInput, err)
		return
	}

	if len(data) == 0 {
		SendError(w, r, http.StatusBadRequest, "Request body is empty", nil)
		return
	}

	insertSQL, args, err := h.builder.BuildInsert(tableName, data)
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidInput, err)
		return
	}

	result, err := h.db.ExecContext(r.Context(), insertSQL, args...)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		lastID = 0
	}

	response := map[string]interface{}{
		"id":      lastID,
		"message": "Record created successfully",
	}

	SendJSON(w, r, http.StatusCreated, response)
}

func (h *GenericHandler) Update(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	id := chi.URLParam(r, "id")

	if !h.schema.TableExists(tableName) {
		SendError(w, r, http.StatusNotFound, ErrTableNotFound, nil)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidInput, err)
		return
	}

	if len(data) == 0 {
		SendError(w, r, http.StatusBadRequest, "Request body is empty", nil)
		return
	}

	updateSQL, args, err := h.builder.BuildUpdate(tableName, id, data)
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidInput, err)
		return
	}

	result, err := h.db.ExecContext(r.Context(), updateSQL, args...)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		SendError(w, r, http.StatusNotFound, ErrRecordNotFound, nil)
		return
	}

	response := map[string]interface{}{
		"message": "Record updated successfully",
		"id":      id,
	}

	SendJSON(w, r, http.StatusOK, response)
}

func (h *GenericHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	id := chi.URLParam(r, "id")

	if !h.schema.TableExists(tableName) {
		SendError(w, r, http.StatusNotFound, ErrTableNotFound, nil)
		return
	}

	deleteSQL, args, err := h.builder.BuildDelete(tableName, id)
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, err)
		return
	}

	result, err := h.db.ExecContext(r.Context(), deleteSQL, args...)
	if err != nil {
		SendError(w, r, http.StatusInternalServerError, ErrDatabaseError, err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		SendError(w, r, http.StatusNotFound, ErrRecordNotFound, nil)
		return
	}

	response := map[string]interface{}{
		"message": "Record deleted successfully",
		"id":      id,
	}

	SendJSON(w, r, http.StatusOK, response)
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))

		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			if val == nil {
				rowMap[col] = nil
				continue
			}

			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else if t, ok := val.(time.Time); ok {
				rowMap[col] = t.Format(time.RFC3339)
			} else {
				rowMap[col] = val
			}
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

type JSONTime time.Time

func (jt JSONTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(jt).Format(time.RFC3339))
}
