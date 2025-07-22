package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tanq16/expenseowl/internal/fx"
)

// JSONStore implementats Storage interface - for JSON file storage
type jsonStore struct {
	configPath string
	filePath   string
	ratesPath  string
	mu         sync.RWMutex
	defaults   map[string]string // allows reusing defaults without querying for config
}

type expensesFileData struct {
	Expenses []Expense `json:"expenses"`
}

func InitializeJsonStore(baseConfig SystemConfig) (*jsonStore, error) {
	configPath := filepath.Join(baseConfig.StorageURL, "config.json")
	filePath := filepath.Join(baseConfig.StorageURL, "expenses.json")
	ratesPath := filepath.Join(baseConfig.StorageURL, "rates.json")
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	// create expenses file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		initialData := expensesFileData{Expenses: []Expense{}}
		data, err := json.Marshal(initialData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal initial data: %v", err)
		}
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to create storage file: %v", err)
		}
		log.Println("Created expense storage file")
	} else {
		log.Println("Found existing expense storage file")
	}

	// create rates file if it doesn't exist
	if _, err := os.Stat(ratesPath); os.IsNotExist(err) {
		initialData := make(map[string]map[string]float64)
		data, err := json.Marshal(initialData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal initial data: %v", err)
		}
		if err := os.WriteFile(ratesPath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to create rates file: %v", err)
		}
		log.Println("Created rates storage file")
	} else {
		log.Println("Found existing rates storage file")
	}

	// create config file if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		initialConfig := Config{}
		initialConfig.SetBaseConfig()
		data, err := json.Marshal(initialConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal initial config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to create config file: %v", err)
		}
		log.Println("Created config storage config")
	} else {
		log.Println("Found existing config storage config")
	}

	return &jsonStore{
		configPath: configPath,
		filePath:   filePath,
		ratesPath:  ratesPath,
		defaults:   map[string]string{},
	}, nil
}

// primitive methods

func (s *jsonStore) readExpensesFile(path string) (*expensesFileData, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data expensesFileData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	log.Println("Read expenses file")
	return &data, nil
}

func (s *jsonStore) writeExpensesFile(path string, data *expensesFileData) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	log.Println("Wrote expenses file")
	return os.WriteFile(path, content, 0644)
}

func (s *jsonStore) readRatesFile(path string) (Rates, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data Rates
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	log.Println("Read rates file")
	return data, nil
}

func (s *jsonStore) writeRatesFile(path string, data Rates) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	log.Println("Wrote rates file")
	return os.WriteFile(path, content, 0644)
}

func (s *jsonStore) readConfigFile(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data Config
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	log.Println("Read config file")
	return &data, nil
}

func (s *jsonStore) writeConfigFile(path string, data *Config) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	log.Println("Wrote config file")
	return os.WriteFile(path, content, 0644)
}

// ------------------------------------------------------------
// JSONStore interface methods
// ------------------------------------------------------------

func (s *jsonStore) Close() error {
	return nil
}

func (s *jsonStore) GetConfig() (*Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readConfigFile(s.configPath)
}

// Basic Config Updates

func (s *jsonStore) GetCategories() ([]string, error) {
	config, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	return config.Categories, nil
}

func (s *jsonStore) UpdateCategories(categories []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readConfigFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	data.Categories = categories
	return s.writeConfigFile(s.configPath, data)
}

func (s *jsonStore) GetCurrencyCatalog() map[string]string {
	return currencyCatalog
}

func (s *jsonStore) GetDefaultCurrency() (string, error) {
	config, err := s.GetConfig()
	if err != nil {
		return "", err
	}
	return config.DefaultCurrency, nil
}

func (s *jsonStore) UpdateDefaultCurrency(currency string) error {
	if !IsValidCurrency(currency) {
		return fmt.Errorf("invalid currency: %s", currency)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readConfigFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	data.DefaultCurrency = currency
	s.defaults["defaultCurrency"] = currency
	return s.writeConfigFile(s.configPath, data)
}

func (s *jsonStore) GetStartDate() (int, error) {
	config, err := s.GetConfig()
	if err != nil {
		return 0, err
	}
	return config.StartDate, nil
}

func (s *jsonStore) UpdateStartDate(startDate int) error {
	if startDate < 1 || startDate > 31 {
		return fmt.Errorf("invalid start date: %d", startDate)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readConfigFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	data.StartDate = startDate
	s.defaults["startDate"] = fmt.Sprintf("%d", startDate)
	return s.writeConfigFile(s.configPath, data)
}

func (s *jsonStore) GetRecurringExpenses() ([]RecurringExpense, error) {
	config, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	return config.RecurringExpenses, nil
}

func (s *jsonStore) GetRecurringExpense(id string) (RecurringExpense, error) {
	recurringExpenses, err := s.GetRecurringExpenses()
	if err != nil {
		return RecurringExpense{}, err
	}
	for _, r := range recurringExpenses {
		if r.ID == id {
			return r, nil
		}
	}
	return RecurringExpense{}, fmt.Errorf("recurring expense with ID %s not found", id)
}

func (s *jsonStore) AddRecurringExpense(recurringExpense RecurringExpense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.readConfigFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	if recurringExpense.ID == "" {
		recurringExpense.ID = uuid.New().String()
	}
	if recurringExpense.Currency == "" {
		recurringExpense.Currency = s.defaults["defaultCurrency"]
	}
	config.RecurringExpenses = append(config.RecurringExpenses, recurringExpense)
	if err := s.writeConfigFile(s.configPath, config); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	expensesToAdd := generateExpensesFromRecurring(recurringExpense, false)
	return s.AddMultipleExpenses(expensesToAdd)
}

func (s *jsonStore) RemoveRecurringExpense(id string, removeAll bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.readConfigFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	var found bool
	var updatedRecurringExpenses []RecurringExpense
	for _, r := range config.RecurringExpenses {
		if r.ID == id {
			found = true
		} else {
			updatedRecurringExpenses = append(updatedRecurringExpenses, r)
		}
	}
	if !found {
		return fmt.Errorf("recurring expense with ID %s not found", id)
	}
	config.RecurringExpenses = updatedRecurringExpenses
	expensesData, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	var updatedExpenses []Expense
	today := time.Now()
	for _, exp := range expensesData.Expenses {
		if exp.RecurringID != id {
			updatedExpenses = append(updatedExpenses, exp)
			continue
		}
		if !removeAll && !exp.Date.After(today) {
			updatedExpenses = append(updatedExpenses, exp)
		}
	}
	expensesData.Expenses = updatedExpenses
	if err := s.writeExpensesFile(s.filePath, expensesData); err != nil {
		return err
	}
	return s.writeConfigFile(s.configPath, config)
}

func (s *jsonStore) UpdateRecurringExpense(id string, recurringExpense RecurringExpense, updateAll bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, err := s.readConfigFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	var found bool
	for i, r := range config.RecurringExpenses {
		if r.ID == id {
			recurringExpense.ID = id // Ensure ID is preserved
			if recurringExpense.Currency == "" {
				recurringExpense.Currency = s.defaults["defaultCurrency"]
			}
			config.RecurringExpenses[i] = recurringExpense
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("recurring expense with ID %s not found", id)
	}
	expensesData, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	var remainingExpenses []Expense
	today := time.Now()
	for _, exp := range expensesData.Expenses {
		if exp.RecurringID != id {
			remainingExpenses = append(remainingExpenses, exp)
			continue
		}
		if !updateAll && !exp.Date.After(today) {
			remainingExpenses = append(remainingExpenses, exp)
		}
	}
	expensesData.Expenses = remainingExpenses
	expensesToAdd := generateExpensesFromRecurring(recurringExpense, !updateAll)
	expensesData.Expenses = append(expensesData.Expenses, expensesToAdd...)
	if err := s.writeExpensesFile(s.filePath, expensesData); err != nil {
		return err
	}
	return s.writeConfigFile(s.configPath, config)
}

// Expenses

func (s *jsonStore) GetAllExpenses() ([]Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage file: %v", err)
	}
	return data.Expenses, nil
}

func (s *jsonStore) GetExpense(id string) (Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return Expense{}, fmt.Errorf("failed to read storage file: %v", err)
	}
	for i, exp := range data.Expenses {
		if exp.ID == id {
			log.Printf("Retrieved expense with ID %s\n", id)
			return data.Expenses[i], nil
		}
	}
	return Expense{}, fmt.Errorf("expense with ID %s not found", id)
}

func (s *jsonStore) AddExpense(expense Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}

	if expense.ID == "" {
		expense.ID = uuid.New().String()
	}
	if expense.Currency == "" {
		expense.Currency = s.defaults["defaultCurrency"]
	}
	if expense.Date.IsZero() {
		expense.Date = time.Now()
	}
	if expense.Currency != s.defaults["defaultCurrency"] {
		go s.fetchRateForExpense(expense)
	}
	data.Expenses = append(data.Expenses, expense)
	log.Printf("Added expense with ID %s\n", expense.ID)
	return s.writeExpensesFile(s.filePath, data)
}

func (s *jsonStore) RemoveExpense(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	found := false
	newExpenses := make([]Expense, 0, len(data.Expenses)-1)
	for _, exp := range data.Expenses {
		if exp.ID != id {
			newExpenses = append(newExpenses, exp)
		} else {
			found = true
		}
	}
	if !found {
		log.Printf("Expense with ID %s not found\n", id)
		return fmt.Errorf("expense with ID %s not found", id)
	}
	log.Printf("Deleted expense with ID %s\n", id)
	data.Expenses = newExpenses
	return s.writeExpensesFile(s.filePath, data)
}

func (s *jsonStore) AddMultipleExpenses(expensesToAdd []Expense) error {
	if len(expensesToAdd) == 0 {
		return nil
	}
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	data.Expenses = append(data.Expenses, expensesToAdd...)
	log.Printf("Added %d new recurring expense instances\n", len(expensesToAdd))
	return s.writeExpensesFile(s.filePath, data)
}

func (s *jsonStore) RemoveMultipleExpenses(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	idsToRemove := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idsToRemove[id] = struct{}{}
	}
	originalCount := len(data.Expenses)
	newExpenses := make([]Expense, 0, originalCount)
	for _, exp := range data.Expenses {
		if _, found := idsToRemove[exp.ID]; !found {
			newExpenses = append(newExpenses, exp)
		}
	}
	if len(newExpenses) == originalCount {
		log.Println("RemoveMultipleExpenses: no expenses found to remove")
		return nil
	}
	log.Printf("Removed %d expenses\n", originalCount-len(newExpenses))
	data.Expenses = newExpenses
	return s.writeExpensesFile(s.filePath, data)
}

func (s *jsonStore) UpdateExpense(id string, expense Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readExpensesFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	found := false
	for i, exp := range data.Expenses {
		if exp.ID == id {
			data.Expenses[i] = expense
			data.Expenses[i].ID = id
			if data.Expenses[i].Currency == "" {
				data.Expenses[i].Currency = s.defaults["defaultCurrency"]
			}
			found = true
			break
		}
	}
	if !found {
		log.Printf("expense with ID %s not found\n", id)
		return fmt.Errorf("expense with ID %s not found", id)
	}
	log.Printf("Edited expense with ID %s\n", id)
	return s.writeExpensesFile(s.filePath, data)
}

func (s *jsonStore) GetRate(day time.Time, base, quote string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.readRatesFile(s.ratesPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read storage file: %v", err)
	}
	dateStr := fx.FormatToString(day)
	var result float64
	result, exist := data[dateStr][base][quote]
	if !exist {
		return 0, fmt.Errorf("rate %s→%s (%s) missing", base, quote, fx.FormatToString(day))
	}
	return result, nil
}

func (s *jsonStore) GetRates(ratesParams map[string]map[string][]string) (Rates, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.readRatesFile(s.ratesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage file: %v", err)
	}

	notFound := make(map[string]map[string]struct{})
	rates := make(Rates, len(ratesParams))

	for day, baseMap := range ratesParams {
		rates[day] = make(map[string]map[string]float64)
		notFound[day] = make(map[string]struct{})
		for base, quotes := range baseMap {
			notFound[day][base] = struct{}{}
			rates[day][base] = make(map[string]float64)
			for _, quote := range quotes {
				if rate, exist := data[day][base][quote]; exist {
					rates[day][base][quote] = rate
					delete(notFound[day], base)
				}
			}
		}
	}

	// ── 3) background fetch for missing rates  ────────────────────────────
	for day, missingBases := range notFound {
		for base := range missingBases {
			dayStr, err := fx.FormatToTime(day)
			if err != nil {
				log.Printf("Error formating date string to time: %v", err)
				return rates, nil
			}
			go s.fetchRateAndUpdateJson(base, dayStr)
		}
	}
	return rates, nil
}

func (s *jsonStore) fetchRateForExpense(expense Expense) error {
	return s.fetchRateAndUpdateJson(expense.Currency, expense.Date)
}

func (s *jsonStore) fetchRateAndUpdateJson(currency string, date time.Time) error {
	rates, err := fx.RatesOn(currency, date)
	if err != nil {
		return fmt.Errorf("failed to get rate: %v", err)
	}
	KeepOnlyValidCurrencies(rates, currencyCatalog)
	return s.updateRates(date, currency, rates)
}

func (s *jsonStore) updateRates(date time.Time, currency string, rates map[string]float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readRatesFile(s.ratesPath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	toUpdate := false
	var dateStr = fx.FormatToString(date)

	if _, exists := data[dateStr]; !exists {
		data[dateStr] = make(map[string]map[string]float64)
	}
	if _, exists := data[dateStr][currency]; !exists {
		data[dateStr][currency] = rates
		toUpdate = true
	}
	if !toUpdate {
		log.Println("No rates to update")
		return nil
	}
	log.Printf("Updated rates with %s@%s\n", currency, dateStr)
	return s.writeRatesFile(s.ratesPath, data)
}
