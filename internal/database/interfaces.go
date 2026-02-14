package database

import (
	"context"
	"database/sql"
)

type Driver interface {
	Connect(ctx context.Context) (*sql.DB, error)

	Ping(ctx context.Context, db *sql.DB) error

	GetTables(ctx context.Context, db *sql.DB) ([]string, error)

	GetColumns(ctx context.Context, db *sql.DB, table string) ([]ColumnInfo, error)

	GetPrimaryKey(ctx context.Context, db *sql.DB, table string) (string, error)

	GetRelationships(ctx context.Context, db *sql.DB, table string) ([]RelationshipInfo, error)
}

type ColumnInfo struct {
	Name            string
	Type            string
	Nullable        bool
	DefaultValue    sql.NullString
	IsPrimaryKey    bool
	IsAutoIncrement bool
	MaxLength       sql.NullInt64
	EnumValues      []string // ENUM değerleri (dinamik)
	IsUnique        bool     // Unique constraint var mı?
}

type RelationshipInfo struct {
	ColumnName         string
	ReferencedTable    string
	ReferencedColumn   string
	ConstraintName     string
}

type ConnectionManager struct {
	db     *sql.DB
	driver Driver
}

func NewConnectionManager(driver Driver) *ConnectionManager {
	return &ConnectionManager{
		driver: driver,
	}
}

func (cm *ConnectionManager) Connect(ctx context.Context) error {
	db, err := cm.driver.Connect(ctx)
	if err != nil {
		return err
	}

	cm.db = db

	if err := cm.driver.Ping(ctx, db); err != nil {
		db.Close()
		return err
	}

	return nil
}

func (cm *ConnectionManager) Close() error {
	if cm.db != nil {
		return cm.db.Close()
	}
	return nil
}

func (cm *ConnectionManager) GetDB() *sql.DB {
	return cm.db
}

func (cm *ConnectionManager) GetDriver() Driver {
	return cm.driver
}
