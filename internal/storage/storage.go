package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Storage interface for all storage types
type Storage interface {
	Initialize() error
	Close() error

	GetConfig() (*Config, error)
	GetCategories() ([]string, error)
	UpdateCategories(categories []string) error
	GetCurrency() (string, error)
	UpdateCurrency(currency string) error
	GetStartDate() (int, error)
	UpdateStartDate(startDate int) error

	SaveExpense(expense *Expense) error
	GetAllExpenses() ([]*Expense, error)
	GetExpense(id string) (*Expense, error)
	DeleteExpense(id string) error
	EditExpense(expense *Expense) error
}

// Initialize initializes the storage
func InitializeStorage() (Storage, error) {
	dataStore := "json"
	overrideDataStore := os.Getenv("DATA_STORE")
	if overrideDataStore != "" {
		dataStore = overrideDataStore
	}
	switch dataStore {
	case "json":
		return InitializeJsonStore()
	case "mysql":
		return InitializeMySqlStore()
	case "postgres":
		return InitializePostgresStore()
	case "sqlite":
		return InitializeSqliteStore()
	}
	return nil, fmt.Errorf("invalid data store: %s", dataStore)
}

// JSONStore implementats Storage interface - for JSON file storage
type jsonStore struct {
	configPath string
	filePath   string
	mu         sync.RWMutex
}

type fileData struct {
	Expenses []*Expense `json:"expenses"`
}

// initializes the JSON storage backend
func InitializeJsonStore() (*jsonStore, error) {
	customDir := os.Getenv("JSON_DIR")
	if customDir == "" {
		customDir = "data"
	}
	configPath := filepath.Join(customDir, "config.json")
	filePath := filepath.Join(customDir, "expenses.json")
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		initialData := fileData{Expenses: []*Expense{}}
		data, err := json.Marshal(initialData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal initial data: %v", err)
		}
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to create storage file: %v", err)
		}
	}
	log.Println("Created expense storage file")
	return &jsonStore{
		configPath: configPath,
		filePath:   filePath,
	}, nil
}

// mySqlStore implements Storage interface - for MySQL database storage
type mySqlStore struct {
	db *sql.DB
}

// initializes the MySQL storage backend
func InitializeMySqlStore() (*mySqlStore, error) {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL database: %v", err)
	}
	return &mySqlStore{db: db}, nil
}

// postgresStore implements Storage interface - for PostgreSQL database storage
type postgresStore struct {
	db *sql.DB
}

// initializes the PostgreSQL storage backend
func InitializePostgresStore() (*postgresStore, error) {
	db, err := sql.Open("postgres", os.Getenv("POSTGRES_DSN"))
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %v", err)
	}
	return &postgresStore{db: db}, nil
}

// sqliteStore implements Storage interface - for SQLite database storage
type sqliteStore struct {
	db *sql.DB
}

// initializes the SQLite storage backend
func InitializeSqliteStore() (*sqliteStore, error) {
	db, err := sql.Open("sqlite3", os.Getenv("SQLITE_DSN"))
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %v", err)
	}
	return &sqliteStore{db: db}, nil
}
