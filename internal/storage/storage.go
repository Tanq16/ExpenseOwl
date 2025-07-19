package storage

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Storage interface for all storage types
type Storage interface {
	Close() error
	GetConfig() (*Config, error)

	// Basic Config Updates
	GetCategories() ([]string, error)
	UpdateCategories(categories []string) error
	// GetTags() ([]string, error)
	// UpdateTags(tags []string) error
	GetCurrency() (string, error)
	UpdateCurrency(currency string) error
	GetStartDate() (int, error)
	UpdateStartDate(startDate int) error

	// Recurring Expenses
	GetRecurringExpenses() ([]RecurringExpense, error)
	GetRecurringExpense(id string) (RecurringExpense, error)
	AddRecurringExpense(recurringExpense RecurringExpense) error
	RemoveRecurringExpense(id string, removeAll bool) error
	UpdateRecurringExpense(id string, recurringExpense RecurringExpense, updateAll bool) error

	// Expenses
	GetAllExpenses() ([]Expense, error)
	GetExpense(id string) (Expense, error)
	AddExpense(expense Expense) error
	RemoveExpense(id string) error
	AddMultipleExpenses(expenses []Expense) error
	RemoveMultipleExpenses(ids []string) error
	UpdateExpense(id string, expense Expense) error

	// Potential Future Feature: Multi-currency
	// GetConversions() (map[string]float64, error)
	// UpdateConversions(conversions map[string]float64) error
}

// config for expense data
type Config struct {
	Categories        []string           `json:"categories"`
	Currency          string             `json:"currency"`
	StartDate         int                `json:"startDate"`
	RecurringExpenses []RecurringExpense `json:"recurringExpenses"`
	// Tags              []string           `json:"tags"`
}

type RecurringExpense struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Tags        []string  `json:"tags"`
	Category    string    `json:"category"`
	StartDate   time.Time `json:"startDate"`   // date of the first occurrence
	Interval    string    `json:"interval"`    // daily, weekly, monthly, yearly
	Occurrences int       `json:"occurrences"` // 0 for 10 years (heuristic), 10 for 10 occurrences
}

type BackendType string

const (
	BackendTypeJSON     BackendType = "json"
	BackendTypePostgres BackendType = "postgres"
)

// config for the storage backend
type SystemConfig struct {
	StorageURL  string
	StorageType BackendType
	StorageUser string
	StoragePass string
}

// expense struct
type Expense struct {
	ID          string    `json:"id"`
	RecurringID string    `json:"recurringID"`
	Name        string    `json:"name"`
	Tags        []string  `json:"tags"`
	Category    string    `json:"category"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Date        time.Time `json:"date"`
}

func (c *Config) SetBaseConfig() {
	c.Categories = defaultCategories
	c.Currency = "usd"
	c.StartDate = 1
	// c.Tags = []string{}
	c.RecurringExpenses = []RecurringExpense{}
}

func (c *SystemConfig) SetStorageConfig() {
	c.StorageType = backendTypeFromEnv(os.Getenv("STORAGE_TYPE"))
	c.StorageURL = backendURLFromEnv(os.Getenv("STORAGE_URL"))
	c.StorageUser = os.Getenv("STORAGE_USER")
	c.StoragePass = os.Getenv("STORAGE_PASS")
}

func backendTypeFromEnv(env string) BackendType {
	switch env {
	case "json":
		return BackendTypeJSON
	case "postgres":
		return BackendTypePostgres
	default:
		return BackendTypeJSON
	}
}

func backendURLFromEnv(env string) string {
	if env == "" {
		return "data"
	}
	return env
}

// initializes the storage backend
func InitializeStorage() (Storage, error) {
	baseConfig := SystemConfig{}
	baseConfig.SetStorageConfig()
	switch baseConfig.StorageType {
	case BackendTypeJSON:
		return InitializeJsonStore(baseConfig)
	case BackendTypePostgres:
		return InitializePostgresStore(baseConfig)
	}
	return nil, fmt.Errorf("invalid data store: %s", baseConfig.StorageType)
}

// cleans a string by replacing < and > with a space and trimms whitespace
func SanitizeString(s string) string {
	s = strings.ReplaceAll(s, "<", " ")
	s = strings.ReplaceAll(s, ">", " ")
	re := regexp.MustCompile(`\s+`) // Collapse multiple spaces into one
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func ValidateCategory(category string) (string, error) {
	sanitized := SanitizeString(category)
	if sanitized == "" {
		return "", fmt.Errorf("category name cannot be empty or contain only invalid characters")
	}
	return sanitized, nil
}

func (e *Expense) Validate() error {
	e.Name = SanitizeString(e.Name)
	if e.Name == "" {
		return fmt.Errorf("expense 'name' cannot be empty")
	}
	if e.Category == "" {
		return fmt.Errorf("expense 'category' cannot be empty")
	}
	if e.Currency == "" {
		return fmt.Errorf("expense 'currency' cannot be empty")
	}
	if len(e.Tags) > 0 {
		var cleanedTags []string
		for _, tag := range e.Tags {
			sanitizedTag := SanitizeString(tag)
			if sanitizedTag != "" {
				cleanedTags = append(cleanedTags, sanitizedTag)
			}
		}
		e.Tags = cleanedTags
	}
	if e.Date.IsZero() {
		return fmt.Errorf("expense 'date' cannot be empty")
	}
	return nil
}

func (e *RecurringExpense) Validate() error {
	e.Name = SanitizeString(e.Name)
	if e.Name == "" {
		return fmt.Errorf("recurring expense 'name' cannot be empty")
	}
	if e.Category == "" {
		return fmt.Errorf("recurring expense 'category' cannot be empty")
	}
	if len(e.Tags) > 0 {
		var cleanedTags []string
		for _, tag := range e.Tags {
			sanitizedTag := SanitizeString(tag)
			if sanitizedTag != "" {
				cleanedTags = append(cleanedTags, sanitizedTag)
			}
		}
		e.Tags = cleanedTags
	}
	if e.Occurrences < 0 {
		return fmt.Errorf("occurrences cannot be a negative number")
	}
	if e.StartDate.IsZero() {
		return fmt.Errorf("start date for recurring expense must be specified")
	}
	validIntervals := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
		"yearly":  true,
	}
	if !validIntervals[e.Interval] {
		return fmt.Errorf("invalid interval: '%s'. Must be one of 'daily', 'weekly', 'monthly', or 'yearly'", e.Interval)
	}
	return nil
}

// variables
var defaultCategories = []string{
	"Food",
	"Groceries",
	"Travel",
	"Rent",
	"Utilities",
	"Entertainment",
	"Healthcare",
	"Shopping",
	"Miscellaneous",
	"Income",
}

var SupportedCurrencies = []string{
	"usd", // US Dollar
	"eur", // Euro
	"gbp", // British Pound
	"jpy", // Japanese Yen
	"cny", // Chinese Yuan
	"krw", // Korean Won
	"inr", // Indian Rupee
	"rub", // Russian Ruble
	"brl", // Brazilian Real
	"zar", // South African Rand
	"aed", // UAE Dirham
	"aud", // Australian Dollar
	"cad", // Canadian Dollar
	"chf", // Swiss Franc
	"hkd", // Hong Kong Dollar
	"sgd", // Singapore Dollar
	"thb", // Thai Baht
	"try", // Turkish Lira
	"mxn", // Mexican Peso
	"php", // Philippine Peso
	"pln", // Polish Złoty
	"sek", // Swedish Krona
	"nzd", // New Zealand Dollar
	"dkk", // Danish Krone
	"idr", // Indonesian Rupiah
	"ils", // Israeli New Shekel
	"vnd", // Vietnamese Dong
	"myr", // Malaysian Ringgit
}
