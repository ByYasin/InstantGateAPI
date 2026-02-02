package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/proyaai/instantgate/internal/config"
	"github.com/proyaai/instantgate/internal/database"
)

type Introspector struct {
	connManager *database.ConnectionManager
	config      *config.DatabaseConfig
	cache       *SchemaCache
	mu          sync.RWMutex
}

func NewIntrospector(cfg *config.DatabaseConfig) *Introspector {
	driver := NewDriver(cfg)
	return &Introspector{
		connManager: database.NewConnectionManager(driver),
		config:      cfg,
		cache:       NewSchemaCache(),
	}
}

func (i *Introspector) Connect(ctx context.Context) error {
	return i.connManager.Connect(ctx)
}

func (i *Introspector) Close() error {
	return i.connManager.Close()
}

func (i *Introspector) GetDB() *sql.DB {
	return i.connManager.GetDB()
}

func (i *Introspector) GetDriver() database.Driver {
	return i.connManager.GetDriver()
}

func (i *Introspector) LoadSchema(ctx context.Context) (*SchemaCache, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	db := i.connManager.GetDB()
	driver := i.connManager.GetDriver()

	tables, err := driver.GetTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	i.cache = NewSchemaCache()

	for _, table := range tables {
		tableSchema, err := i.loadTableSchema(ctx, db, driver, table)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema for table %s: %w", table, err)
		}
		i.cache.Set(table, tableSchema)
	}

	return i.cache, nil
}

func (i *Introspector) loadTableSchema(ctx context.Context, db *sql.DB, driver database.Driver, tableName string) (*TableSchema, error) {
	columns, err := driver.GetColumns(ctx, db, tableName)
	if err != nil {
		return nil, err
	}

	pk, err := driver.GetPrimaryKey(ctx, db, tableName)
	if err != nil {
		return nil, err
	}

	relationships, err := driver.GetRelationships(ctx, db, tableName)
	if err != nil {
		return nil, err
	}

	// Store columns with lowercase keys for case-insensitive lookup
	columnMap := make(map[string]ColumnInfo)
	for _, col := range columns {
		lowerName := strings.ToLower(col.Name)
		columnMap[lowerName] = ColumnInfo{
			Name:            col.Name, // Keep original case for display
			Type:            col.Type,
			GoType:          getGoType(col.Type),
			Nullable:        col.Nullable,
			IsPrimaryKey:    col.IsPrimaryKey,
			IsAutoIncrement: col.IsAutoIncrement,
			MaxLength:       col.MaxLength,
		}
	}

	return &TableSchema{
		Name:          tableName,
		Columns:       columnMap,
		PrimaryKey:    strings.ToLower(pk), // Store lowercase PK
		Relationships: relationships,
	}, nil
}

func (i *Introspector) GetCachedSchema() *SchemaCache {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.cache
}

func (i *Introspector) ReloadSchema(ctx context.Context) (*SchemaCache, error) {
	return i.LoadSchema(ctx)
}

type SchemaCache struct {
	tables map[string]*TableSchema
	mu     sync.RWMutex
}

func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		tables: make(map[string]*TableSchema),
	}
}

func (sc *SchemaCache) Set(table string, schema *TableSchema) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	// Store with lowercase key for case-insensitive lookup
	sc.tables[strings.ToLower(table)] = schema
}

func (sc *SchemaCache) Get(table string) (*TableSchema, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	schema, ok := sc.tables[strings.ToLower(table)]
	return schema, ok
}

func (sc *SchemaCache) GetAll() map[string]*TableSchema {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make(map[string]*TableSchema, len(sc.tables))
	for k, v := range sc.tables {
		result[k] = v
	}
	return result
}

func (sc *SchemaCache) GetTables() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	tables := make([]string, 0, len(sc.tables))
	for table := range sc.tables {
		tables = append(tables, table)
	}
	return tables
}

func (sc *SchemaCache) TableExists(table string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	_, ok := sc.tables[strings.ToLower(table)]
	return ok
}

type TableSchema struct {
	Name          string
	Columns       map[string]ColumnInfo
	PrimaryKey    string
	Relationships []database.RelationshipInfo
}

type ColumnInfo struct {
	Name            string
	Type            string
	GoType          string
	Nullable        bool
	IsPrimaryKey    bool
	IsAutoIncrement bool
	MaxLength       sql.NullInt64
}

func getGoType(mysqlType string) string {
	baseType := ParseType(mysqlType)
	mapping, ok := GetTypeMapping(baseType)
	if !ok {
		return "interface{}"
	}
	return mapping.GoType
}
