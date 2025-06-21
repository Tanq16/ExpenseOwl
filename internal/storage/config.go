package storage

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	ServerPort  string
	StoragePath string
	Categories  []string
	Currency    string
	StartDate   int
	mu          sync.RWMutex
}

type FileConfig struct {
	Categories []string `json:"categories"`
	Currency   string   `json:"currency"`
	StartDate  int      `json:"startDate"`
}

type Expense struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Category string    `json:"category"`
	Amount   float64   `json:"amount"`
	Date     time.Time `json:"date"`
}

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

var currencySymbols = map[string]string{
	"usd": "$",    // US Dollar
	"eur": "€",    // Euro
	"gbp": "£",    // British Pound
	"jpy": "¥",    // Japanese Yen
	"cny": "¥",    // Chinese Yuan
	"krw": "₩",    // Korean Won
	"inr": "₹",    // Indian Rupee
	"rub": "₽",    // Russian Ruble
	"brl": "R$",   // Brazilian Real
	"zar": "R",    // South African Rand
	"aed": "AED",  // UAE Dirham
	"aud": "A$",   // Australian Dollar
	"cad": "C$",   // Canadian Dollar
	"chf": "Fr",   // Swiss Franc
	"hkd": "HK$",  // Hong Kong Dollar
	"sgd": "S$",   // Singapore Dollar
	"thb": "฿",    // Thai Baht
	"try": "₺",    // Turkish Lira
	"mxn": "Mex$", // Mexican Peso
	"php": "₱",    // Philippine Peso
	"pln": "zł",   // Polish Złoty
	"sek": "kr",   // Swedish Krona
	"nzd": "NZ$",  // New Zealand Dollar
	"dkk": "kr.",  // Danish Krone
	"idr": "Rp",   // Indonesian Rupiah
	"ils": "₪",    // Israeli New Shekel
	"vnd": "₫",    // Vietnamese Dong
	"myr": "RM",   // Malaysian Ringgit
}

func (e *Expense) Validate() error {
	if e.Name == "" {
		return errors.New("expense name is required")
	}
	if e.Category == "" {
		return errors.New("category is required")
	}
	if e.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}

func NewConfig(dataPath string) *Config {
	finalPath := ""
	if dataPath == "data" {
		finalPath = filepath.Join(".", "data")
	} else {
		finalPath = filepath.Clean(dataPath)
	}
	if err := os.MkdirAll(finalPath, 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
	}
	log.Printf("Using data directory: %s\n", finalPath)
	cfg := &Config{
		ServerPort:  "8080",
		StoragePath: finalPath,
		Categories:  defaultCategories,
		StartDate:   1,
		Currency:    "$", // Default to USD
	}
	configPath := filepath.Join(finalPath, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Println("Configuration file not found, creating default configuration")
		if envCategories := os.Getenv("EXPENSE_CATEGORIES"); envCategories != "" {
			categories := strings.Split(envCategories, ",")
			for i := range categories {
				categories[i] = strings.TrimSpace(categories[i])
			}
			cfg.Categories = categories
			log.Println("Using custom categories from environment variables")
		}
		if envCurrency := strings.ToLower(os.Getenv("CURRENCY")); envCurrency != "" {
			if symbol, exists := currencySymbols[envCurrency]; exists {
				cfg.Currency = symbol
			}
			log.Println("Using custom currency from environment variables")
		}
		if envStartDate := strings.ToLower(os.Getenv("START_DATE")); envStartDate != "" {
			startDate, err := strconv.Atoi(envStartDate)
			if err != nil {
				log.Println("START_DATE is not a number, using default (1)")
			} else {
				cfg.StartDate = startDate
				log.Println("using custom start date from environment variables")
			}
		}
	} else if fileConfig, err := loadConfigFile(configPath); err == nil {
		cfg.Categories = fileConfig.Categories
		cfg.Currency = fileConfig.Currency
		cfg.StartDate = fileConfig.StartDate
		log.Println("Loaded configuration from file")
	}
	cfg.SaveConfig()
	return cfg
}

func loadConfigFile(filePath string) (*FileConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var config FileConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) SaveConfig() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	filePath := filepath.Join(c.StoragePath, "config.json")
	fileConfig := FileConfig{
		Categories: c.Categories,
		Currency:   c.Currency,
		StartDate:  c.StartDate,
	}
	data, err := json.MarshalIndent(fileConfig, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (c *Config) UpdateCategories(categories []string) error {
	c.mu.Lock()
	c.Categories = categories
	c.mu.Unlock()
	return c.SaveConfig()
}

func (c *Config) UpdateCurrency(currencyCode string) error {
	c.mu.Lock()
	if symbol, exists := currencySymbols[strings.ToLower(currencyCode)]; exists {
		c.Currency = symbol
	} else {
		c.mu.Unlock()
		return errors.New("invalid currency code")
	}
	c.mu.Unlock()
	return c.SaveConfig()
}

func (c *Config) UpdateStartDate(startDate int) error {
	c.mu.Lock()
	c.StartDate = max(min(startDate, 31), 1)
	c.mu.Unlock()
	return c.SaveConfig()
}
