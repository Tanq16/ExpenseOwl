package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// databaseStore implements the Storage interface for PostgreSQL.
type databaseStore struct {
	db *sql.DB
}

// SQL queries as constants for reusability and clarity.
const (
	createExpensesTableSQL = `
	CREATE TABLE IF NOT EXISTS expenses (
		id VARCHAR(36) PRIMARY KEY,
		recurring_id VARCHAR(36),
		name VARCHAR(255) NOT NULL,
		category VARCHAR(255) NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		date TIMESTAMPTZ NOT NULL,
		tags TEXT
	);`

	createRecurringExpensesTableSQL = `
	CREATE TABLE IF NOT EXISTS recurring_expenses (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		category VARCHAR(255) NOT NULL,
		start_date TIMESTAMPTZ NOT NULL,
		interval VARCHAR(50) NOT NULL,
		occurrences INTEGER NOT NULL,
		tags TEXT
	);`

	createConfigTableSQL = `
	CREATE TABLE IF NOT EXISTS config (
		id VARCHAR(255) PRIMARY KEY DEFAULT 'default',
		categories TEXT NOT NULL,
		currency VARCHAR(255) NOT NULL,
		start_date INTEGER NOT NULL
	);`
)

func InitializePostgresStore(baseConfig SystemConfig) (Storage, error) {
	dbURL := makeDBURL(baseConfig)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %v", err)
	}
	log.Println("Connected to PostgreSQL database")

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create database tables: %v", err)
	}

	return &databaseStore{db: db}, nil
}

func makeDBURL(baseConfig SystemConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s?sslmode=disable", baseConfig.StorageUser, baseConfig.StoragePass, baseConfig.StorageURL)
}

func createTables(db *sql.DB) error {
	for _, query := range []string{createExpensesTableSQL, createRecurringExpensesTableSQL, createConfigTableSQL} {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (s *databaseStore) Close() error {
	return s.db.Close()
}

func (s *databaseStore) saveConfig(config *Config) error {
	categoriesJSON, err := json.Marshal(config.Categories)
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %v", err)
	}

	query := `
		INSERT INTO config (id, categories, currency, start_date)
		VALUES ('default', $1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			categories = EXCLUDED.categories,
			currency = EXCLUDED.currency,
			start_date = EXCLUDED.start_date;
	`
	_, err = s.db.Exec(query, string(categoriesJSON), config.Currency, config.StartDate)
	return err
}

func (s *databaseStore) updateConfig(updater func(c *Config) error) error {
	config, err := s.GetConfig()
	if err != nil {
		return err
	}
	if err := updater(config); err != nil {
		return err
	}
	return s.saveConfig(config)
}

func (s *databaseStore) GetConfig() (*Config, error) {
	query := `SELECT categories, currency, start_date FROM config WHERE id = 'default'`
	var categoriesStr, currency string
	var startDate int
	err := s.db.QueryRow(query).Scan(&categoriesStr, &currency, &startDate)

	if err != nil {
		if err == sql.ErrNoRows {
			config := &Config{}
			config.SetBaseConfig()
			if err := s.saveConfig(config); err != nil {
				return nil, fmt.Errorf("failed to save initial default config: %v", err)
			}
			return config, nil
		}
		return nil, fmt.Errorf("failed to get config from db: %v", err)
	}

	var config Config
	config.Currency = currency
	config.StartDate = startDate
	if err := json.Unmarshal([]byte(categoriesStr), &config.Categories); err != nil {
		return nil, fmt.Errorf("failed to parse categories from db: %v", err)
	}

	recurring, err := s.GetRecurringExpenses()
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring expenses for config: %v", err)
	}
	config.RecurringExpenses = recurring

	return &config, nil
}

func (s *databaseStore) GetCategories() ([]string, error) {
	config, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	return config.Categories, nil
}

func (s *databaseStore) UpdateCategories(categories []string) error {
	return s.updateConfig(func(c *Config) error {
		c.Categories = categories
		return nil
	})
}

func (s *databaseStore) GetCurrency() (string, error) {
	config, err := s.GetConfig()
	if err != nil {
		return "", err
	}
	return config.Currency, nil
}

func (s *databaseStore) UpdateCurrency(currency string) error {
	if !slices.Contains(supportedCurrencies, currency) {
		return fmt.Errorf("invalid currency: %s", currency)
	}
	return s.updateConfig(func(c *Config) error {
		c.Currency = currency
		return nil
	})
}

func (s *databaseStore) GetStartDate() (int, error) {
	config, err := s.GetConfig()
	if err != nil {
		return 0, err
	}
	return config.StartDate, nil
}

func (s *databaseStore) UpdateStartDate(startDate int) error {
	if startDate < 1 || startDate > 31 {
		return fmt.Errorf("invalid start date: %d", startDate)
	}
	return s.updateConfig(func(c *Config) error {
		c.StartDate = startDate
		return nil
	})
}

func scanExpense(scanner interface{ Scan(...any) error }) (Expense, error) {
	var expense Expense
	var tagsStr sql.NullString
	var recurringID sql.NullString
	err := scanner.Scan(&expense.ID, &recurringID, &expense.Name, &expense.Category, &expense.Amount, &expense.Date, &tagsStr)
	if err != nil {
		return Expense{}, err
	}
	if recurringID.Valid {
		expense.RecurringID = recurringID.String
	}
	if tagsStr.Valid && tagsStr.String != "" {
		if err := json.Unmarshal([]byte(tagsStr.String), &expense.Tags); err != nil {
			return Expense{}, fmt.Errorf("failed to parse tags for expense %s: %v", expense.ID, err)
		}
	}
	return expense, nil
}

func (s *databaseStore) GetAllExpenses() ([]Expense, error) {
	query := `SELECT id, recurring_id, name, category, amount, date, tags FROM expenses ORDER BY date DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenses: %v", err)
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		expense, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expense: %v", err)
		}
		expenses = append(expenses, expense)
	}
	return expenses, nil
}

func (s *databaseStore) GetExpense(id string) (Expense, error) {
	query := `SELECT id, recurring_id, name, category, amount, date, tags FROM expenses WHERE id = $1`
	expense, err := scanExpense(s.db.QueryRow(query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return Expense{}, fmt.Errorf("expense with ID %s not found", id)
		}
		return Expense{}, fmt.Errorf("failed to get expense: %v", err)
	}
	return expense, nil
}

func (s *databaseStore) AddExpense(expense Expense) error {
	if expense.ID == "" {
		expense.ID = uuid.New().String()
	}
	tagsJSON, err := json.Marshal(expense.Tags)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO expenses (id, recurring_id, name, category, amount, date, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = s.db.Exec(query, expense.ID, expense.RecurringID, expense.Name, expense.Category, expense.Amount, expense.Date, string(tagsJSON))
	return err
}

func (s *databaseStore) UpdateExpense(id string, expense Expense) error {
	tagsJSON, err := json.Marshal(expense.Tags)
	if err != nil {
		return err
	}
	query := `
		UPDATE expenses
		SET name = $1, category = $2, amount = $3, date = $4, tags = $5, recurring_id = $6
		WHERE id = $7
	`
	result, err := s.db.Exec(query, expense.Name, expense.Category, expense.Amount, expense.Date, string(tagsJSON), expense.RecurringID, id)
	if err != nil {
		return fmt.Errorf("failed to update expense: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("expense with ID %s not found", id)
	}
	return nil
}

func (s *databaseStore) RemoveExpense(id string) error {
	query := `DELETE FROM expenses WHERE id = $1`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete expense: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("expense with ID %s not found", id)
	}
	return nil
}

func (s *databaseStore) AddMultipleExpenses(expenses []Expense) error {
	if len(expenses) == 0 {
		return nil
	}
	// use the same addexpense method
	for _, exp := range expenses {
		if err := s.AddExpense(exp); err != nil {
			return err
		}
	}
	return nil
}

func (s *databaseStore) RemoveMultipleExpenses(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `DELETE FROM expenses WHERE id = ANY($1)`
	_, err := s.db.Exec(query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete multiple expenses: %v", err)
	}
	return nil
}

func scanRecurringExpense(scanner interface{ Scan(...interface{}) error }) (RecurringExpense, error) {
	var re RecurringExpense
	var tagsStr sql.NullString
	err := scanner.Scan(&re.ID, &re.Name, &re.Amount, &re.Category, &re.StartDate, &re.Interval, &re.Occurrences, &tagsStr)
	if err != nil {
		return RecurringExpense{}, err
	}
	if tagsStr.Valid && tagsStr.String != "" {
		if err := json.Unmarshal([]byte(tagsStr.String), &re.Tags); err != nil {
			return RecurringExpense{}, fmt.Errorf("failed to parse tags for recurring expense %s: %v", re.ID, err)
		}
	}
	return re, nil
}

func (s *databaseStore) GetRecurringExpenses() ([]RecurringExpense, error) {
	query := `SELECT id, name, amount, category, start_date, interval, occurrences, tags FROM recurring_expenses`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query recurring expenses: %v", err)
	}
	defer rows.Close()
	var recurringExpenses []RecurringExpense
	for rows.Next() {
		re, err := scanRecurringExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recurring expense: %v", err)
		}
		recurringExpenses = append(recurringExpenses, re)
	}
	return recurringExpenses, nil
}

func (s *databaseStore) GetRecurringExpense(id string) (RecurringExpense, error) {
	query := `SELECT id, name, amount, category, start_date, interval, occurrences, tags FROM recurring_expenses WHERE id = $1`
	re, err := scanRecurringExpense(s.db.QueryRow(query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return RecurringExpense{}, fmt.Errorf("recurring expense with ID %s not found", id)
		}
		return RecurringExpense{}, fmt.Errorf("failed to get recurring expense: %v", err)
	}
	return re, nil
}

func (s *databaseStore) AddRecurringExpense(recurringExpense RecurringExpense) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // Rollback on error

	if recurringExpense.ID == "" {
		recurringExpense.ID = uuid.New().String()
	}
	tagsJSON, _ := json.Marshal(recurringExpense.Tags)
	ruleQuery := `
		INSERT INTO recurring_expenses (id, name, amount, category, start_date, interval, occurrences, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ruleQuery, recurringExpense.ID, recurringExpense.Name, recurringExpense.Amount, recurringExpense.Category, recurringExpense.StartDate, recurringExpense.Interval, recurringExpense.Occurrences, string(tagsJSON))
	if err != nil {
		return fmt.Errorf("failed to insert recurring expense rule: %v", err)
	}

	expensesToAdd := generateExpensesFromRecurring(recurringExpense, false)
	if len(expensesToAdd) > 0 {
		stmt, err := tx.Prepare(pq.CopyIn("expenses", "id", "recurring_id", "name", "category", "amount", "date", "tags"))
		if err != nil {
			return fmt.Errorf("failed to prepare copy in: %v", err)
		}
		defer stmt.Close()
		for _, exp := range expensesToAdd {
			expTagsJSON, _ := json.Marshal(exp.Tags)
			_, err = stmt.Exec(exp.ID, exp.RecurringID, exp.Name, exp.Category, exp.Amount, exp.Date, string(expTagsJSON))
			if err != nil {
				return fmt.Errorf("failed to execute copy in: %v", err)
			}
		}
		if _, err = stmt.Exec(); err != nil {
			return fmt.Errorf("failed to finalize copy in: %v", err)
		}
	}
	return tx.Commit()
}

func (s *databaseStore) UpdateRecurringExpense(id string, recurringExpense RecurringExpense, updateAll bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()
	recurringExpense.ID = id // Ensure ID is preserved
	tagsJSON, _ := json.Marshal(recurringExpense.Tags)
	ruleQuery := `
		UPDATE recurring_expenses
		SET name = $1, amount = $2, category = $3, start_date = $4, interval = $5, occurrences = $6, tags = $7
		WHERE id = $8
	`
	res, err := tx.Exec(ruleQuery, recurringExpense.Name, recurringExpense.Amount, recurringExpense.Category, recurringExpense.StartDate, recurringExpense.Interval, recurringExpense.Occurrences, string(tagsJSON), id)
	if err != nil {
		return fmt.Errorf("failed to update recurring expense rule: %v", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("recurring expense with ID %s not found to update", id)
	}

	var deleteQuery string
	if updateAll {
		deleteQuery = `DELETE FROM expenses WHERE recurring_id = $1`
		_, err = tx.Exec(deleteQuery, id)
	} else {
		deleteQuery = `DELETE FROM expenses WHERE recurring_id = $1 AND date > $2`
		_, err = tx.Exec(deleteQuery, id, time.Now())
	}
	if err != nil {
		return fmt.Errorf("failed to delete old expense instances for update: %v", err)
	}

	expensesToAdd := generateExpensesFromRecurring(recurringExpense, !updateAll)
	if len(expensesToAdd) > 0 {
		stmt, err := tx.Prepare(pq.CopyIn("expenses", "id", "recurring_id", "name", "category", "amount", "date", "tags"))
		if err != nil {
			return fmt.Errorf("failed to prepare copy in for update: %v", err)
		}
		defer stmt.Close()
		for _, exp := range expensesToAdd {
			expTagsJSON, _ := json.Marshal(exp.Tags)
			_, err = stmt.Exec(exp.ID, exp.RecurringID, exp.Name, exp.Category, exp.Amount, exp.Date, string(expTagsJSON))
			if err != nil {
				return fmt.Errorf("failed to execute copy in for update: %v", err)
			}
		}
		if _, err = stmt.Exec(); err != nil {
			return fmt.Errorf("failed to finalize copy in for update: %v", err)
		}
	}
	return tx.Commit()
}

func (s *databaseStore) RemoveRecurringExpense(id string, removeAll bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()
	res, err := tx.Exec(`DELETE FROM recurring_expenses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete recurring expense rule: %v", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("recurring expense with ID %s not found", id)
	}

	var deleteQuery string
	if removeAll {
		deleteQuery = `DELETE FROM expenses WHERE recurring_id = $1`
		_, err = tx.Exec(deleteQuery, id)
	} else {
		deleteQuery = `DELETE FROM expenses WHERE recurring_id = $1 AND date > $2`
		_, err = tx.Exec(deleteQuery, id, time.Now())
	}
	if err != nil {
		return fmt.Errorf("failed to delete expense instances: %v", err)
	}
	return tx.Commit()
}

func generateExpensesFromRecurring(recExp RecurringExpense, fromToday bool) []Expense {
	var expenses []Expense
	currentDate := recExp.StartDate
	today := time.Now()
	occurrencesToGenerate := recExp.Occurrences
	if fromToday {
		for currentDate.Before(today) && (recExp.Occurrences == 0 || occurrencesToGenerate > 0) {
			switch recExp.Interval {
			case "daily":
				currentDate = currentDate.AddDate(0, 0, 1)
			case "weekly":
				currentDate = currentDate.AddDate(0, 0, 7)
			case "monthly":
				currentDate = currentDate.AddDate(0, 1, 0)
			case "yearly":
				currentDate = currentDate.AddDate(1, 0, 0)
			default:
				return expenses // Stop if interval is invalid
			}
			if recExp.Occurrences > 0 {
				occurrencesToGenerate--
			}
		}
	}
	limit := occurrencesToGenerate
	if recExp.Occurrences == 0 {
		limit = 365 * 10 // Heuristic for "indefinite"
	}

	for i := 0; i < limit; i++ {
		expense := Expense{
			ID:          uuid.New().String(),
			RecurringID: recExp.ID,
			Name:        recExp.Name,
			Category:    recExp.Category,
			Amount:      recExp.Amount,
			Date:        currentDate,
			Tags:        recExp.Tags,
		}
		expenses = append(expenses, expense)
		switch recExp.Interval {
		case "daily":
			currentDate = currentDate.AddDate(0, 0, 1)
		case "weekly":
			currentDate = currentDate.AddDate(0, 0, 7)
		case "monthly":
			currentDate = currentDate.AddDate(0, 1, 0)
		case "yearly":
			currentDate = currentDate.AddDate(1, 0, 0)
		default:
			return expenses
		}
	}
	return expenses
}
