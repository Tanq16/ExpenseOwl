package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/tanq16/expenseowl/internal/fx"
)

// databaseStore implements the Storage interface for PostgreSQL.
type databaseStore struct {
	db       *sql.DB
	defaults map[string]string // allows reusing defaults without querying for config
}

// SQL queries as constants for reusability and clarity.
const (
	createExpensesTableSQL = `
	CREATE TABLE IF NOT EXISTS expenses (
		id VARCHAR(36) PRIMARY KEY,
		recurring_id VARCHAR(36),
		name VARCHAR(255) NOT NULL,
		category VARCHAR(255) NOT NULL,
		amount NUMERIC(18, 2) NOT NULL,
		currency VARCHAR(3) NOT NULL,
		date TIMESTAMPTZ NOT NULL,
		tags TEXT
	);`

	createRecurringExpensesTableSQL = `
	CREATE TABLE IF NOT EXISTS recurring_expenses (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		amount NUMERIC(18, 2) NOT NULL,
		currency VARCHAR(3) NOT NULL,
		category VARCHAR(255) NOT NULL,
		start_date TIMESTAMPTZ NOT NULL,
		interval VARCHAR(50) NOT NULL,
		occurrences INTEGER NOT NULL,
		tags TEXT
	);`

	createRatesTableSQL = `
	CREATE TABLE IF NOT EXISTS fx_rates (
		day DATE NOT NULL,
		base_currency VARCHAR(3) NOT NULL,
		quote_currency VARCHAR(3) NOT NULL,
		rate NUMERIC(18, 6) NOT NULL, 
		PRIMARY KEY (day, base_currency, quote_currency)
	);`

	createConfigTableSQL = `
	CREATE TABLE IF NOT EXISTS config (
		id VARCHAR(255) PRIMARY KEY DEFAULT 'default',
		categories TEXT NOT NULL,
		default_currency VARCHAR(255) NOT NULL,
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
	return &databaseStore{db: db, defaults: map[string]string{}}, nil
}

func makeDBURL(baseConfig SystemConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s?sslmode=%s", baseConfig.StorageUser, baseConfig.StoragePass, baseConfig.StorageURL, baseConfig.StorageSSL)
}

func createTables(db *sql.DB) error {
	for _, query := range []string{createExpensesTableSQL, createRecurringExpensesTableSQL, createRatesTableSQL, createConfigTableSQL} {
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
		INSERT INTO config (id, categories, default_currency, start_date)
		VALUES ('default', $1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			categories = EXCLUDED.categories,
			default_currency = EXCLUDED.default_currency,
			start_date = EXCLUDED.start_date;
	`
	_, err = s.db.Exec(query, string(categoriesJSON), config.DefaultCurrency, config.StartDate)
	s.defaults["defaultCurrency"] = config.DefaultCurrency
	s.defaults["startDate"] = fmt.Sprintf("%d", config.StartDate)
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
	query := `SELECT categories, default_currency, start_date FROM config WHERE id = 'default'`
	var categoriesStr, defaultCurrency string
	var startDate int
	err := s.db.QueryRow(query).Scan(&categoriesStr, &defaultCurrency, &startDate)

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
	config.DefaultCurrency = defaultCurrency
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

func (s *databaseStore) GetCurrencyCatalog() map[string]string {
	return currencyCatalog
}

func (s *databaseStore) GetDefaultCurrency() (string, error) {
	config, err := s.GetConfig()
	if err != nil {
		return "", err
	}
	return config.DefaultCurrency, nil
}

func (s *databaseStore) UpdateDefaultCurrency(currency string) error {
	if !IsValidCurrency(currency) {
		return fmt.Errorf("invalid currency: %s", currency)
	}
	return s.updateConfig(func(c *Config) error {
		c.DefaultCurrency = currency
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
	err := scanner.Scan(&expense.ID, &recurringID, &expense.Name, &expense.Category, &expense.Amount, &expense.Date, &tagsStr, &expense.Currency)
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
	query := `SELECT id, recurring_id, name, category, amount, date, tags, currency FROM expenses ORDER BY date DESC`
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
	query := `SELECT id, recurring_id, name, category, amount, date, tags, currency FROM expenses WHERE id = $1`
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
	if expense.Currency == "" {
		expense.Currency = s.defaults["defaultCurrency"]
	}
	if expense.Date.IsZero() {
		expense.Date = time.Now()
	}
	if expense.Currency != s.defaults["defaultCurrency"] {
		go s.fetchRatesAndUpdateTable(expense.Currency, expense.Date)
	}
	tagsJSON, err := json.Marshal(expense.Tags)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO expenses (id, recurring_id, name, category, amount, currency, date, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = s.db.Exec(query, expense.ID, expense.RecurringID, expense.Name, expense.Category, expense.Amount, expense.Currency, expense.Date, string(tagsJSON))
	return err
}

func (s *databaseStore) UpdateExpense(id string, expense Expense) error {
	tagsJSON, err := json.Marshal(expense.Tags)
	if err != nil {
		return err
	}
	// TODO: revisit to maybe remove this later, might not be a good default for update
	if expense.Currency == "" {
		expense.Currency = s.defaults["defaultCurrency"]
	}
	query := `
		UPDATE expenses
		SET name = $1, category = $2, amount = $3, currency = $4, date = $5, tags = $6, recurring_id = $7
		WHERE id = $8
	`
	result, err := s.db.Exec(query, expense.Name, expense.Category, expense.Amount, expense.Currency, expense.Date, string(tagsJSON), expense.RecurringID, id)
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

func scanRecurringExpense(scanner interface{ Scan(...any) error }) (RecurringExpense, error) {
	var re RecurringExpense
	var tagsStr sql.NullString
	err := scanner.Scan(&re.ID, &re.Name, &re.Amount, &re.Currency, &re.Category, &re.StartDate, &re.Interval, &re.Occurrences, &tagsStr)
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
	query := `SELECT id, name, amount, currency, category, start_date, interval, occurrences, tags FROM recurring_expenses`
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
	query := `SELECT id, name, amount, category, start_date, interval, occurrences, tags, currency FROM recurring_expenses WHERE id = $1`
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
	if recurringExpense.Currency == "" {
		recurringExpense.Currency = s.defaults["defaultCurrency"]
	}
	tagsJSON, _ := json.Marshal(recurringExpense.Tags)
	ruleQuery := `
		INSERT INTO recurring_expenses (id, name, amount, currency, category, start_date, interval, occurrences, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.Exec(ruleQuery, recurringExpense.ID, recurringExpense.Name, recurringExpense.Amount, recurringExpense.Currency, recurringExpense.Category, recurringExpense.StartDate, recurringExpense.Interval, recurringExpense.Occurrences, string(tagsJSON))
	if err != nil {
		return fmt.Errorf("failed to insert recurring expense rule: %v", err)
	}

	expensesToAdd := generateExpensesFromRecurring(recurringExpense, false)
	if len(expensesToAdd) > 0 {
		stmt, err := tx.Prepare(pq.CopyIn("expenses", "id", "recurring_id", "name", "category", "amount", "currency", "date", "tags"))
		if err != nil {
			return fmt.Errorf("failed to prepare copy in: %v", err)
		}
		defer stmt.Close()
		for _, exp := range expensesToAdd {
			expTagsJSON, _ := json.Marshal(exp.Tags)
			_, err = stmt.Exec(exp.ID, exp.RecurringID, exp.Name, exp.Category, exp.Amount, exp.Currency, exp.Date, string(expTagsJSON))
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
	if recurringExpense.Currency == "" {
		recurringExpense.Currency = s.defaults["defaultCurrency"]
	}
	tagsJSON, _ := json.Marshal(recurringExpense.Tags)
	ruleQuery := `
		UPDATE recurring_expenses
		SET name = $1, amount = $2, category = $3, start_date = $4, interval = $5, occurrences = $6, tags = $7, currency = $8
		WHERE id = $9
	`
	res, err := tx.Exec(ruleQuery, recurringExpense.Name, recurringExpense.Amount, recurringExpense.Category, recurringExpense.StartDate, recurringExpense.Interval, recurringExpense.Occurrences, string(tagsJSON), recurringExpense.Currency, id)
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
		stmt, err := tx.Prepare(pq.CopyIn("expenses", "id", "recurring_id", "name", "category", "amount", "currency", "date", "tags"))
		if err != nil {
			return fmt.Errorf("failed to prepare copy in for update: %v", err)
		}
		defer stmt.Close()
		for _, exp := range expensesToAdd {
			expTagsJSON, _ := json.Marshal(exp.Tags)
			_, err = stmt.Exec(exp.ID, exp.RecurringID, exp.Name, exp.Category, exp.Amount, exp.Currency, exp.Date, string(expTagsJSON))
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
	// if recExp.Occurrences == 0 {
	// 	limit = 2000 // Heuristic for "indefinite"
	// }

	for range limit {
		expense := Expense{
			ID:          uuid.New().String(),
			RecurringID: recExp.ID,
			Name:        recExp.Name,
			Category:    recExp.Category,
			Currency:    recExp.Currency,
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

func (s *databaseStore) GetRate(day time.Time, base, quote string) (float64, error) {
	const query = `
		SELECT rate
		FROM fx_rates
		WHERE day            = $1::date
		AND base_currency  = $2
		AND quote_currency = $3
		LIMIT 1;`
	// base = quote → ratio 1
	if strings.EqualFold(base, quote) {
		return 1, nil
	}

	var result float64
	err := s.db.QueryRow(query, day, strings.ToUpper(base), strings.ToUpper(quote)).Scan(&result)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("rate %s→%s (%s) missing", base, quote, fx.FormatToString(day))
	}
	return result, err
}

func (s *databaseStore) GetAllRates() (Rates, error) {
	query := `
	  SELECT day::date,
	         base_currency,
	         quote_currency,
	         rate
	    FROM fx_rates
	   	ORDER BY day DESC;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rates: %v", err)
	}
	defer rows.Close()

	rates := make(Rates)
	for rows.Next() {
		var day time.Time
		var base, quote string
		var rate float64

		if err := rows.Scan(&day, &base, &quote, &rate); err != nil {
			return nil, fmt.Errorf("failed to query rates: %v", err)
		}

		dateStr := fx.FormatToString(day)
		if _, exists := rates[dateStr]; !exists {
			rates[dateStr] = make(map[string]map[string]float64)
		}
		if _, exists := rates[dateStr][base]; !exists {
			rates[dateStr][base] = make(map[string]float64)
		}
		rates[dateStr][base][quote] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query rates: %v", err)
	}
	return rates, nil
}

func (s *databaseStore) GetRates(ratesParams map[string]map[string][]string) (Rates, error) {
	days := make([]string, 0)
	bases := make([]string, 0)
	quotes := make([]string, 0)

	notFound := make(map[string]map[string]struct{}) // List all "wanted" currencies to fetch from internet.

	for day, baseMap := range ratesParams {
		notFound[day] = make(map[string]struct{})
		for base, quoteList := range baseMap {
			base = strings.ToUpper(base)
			notFound[day][base] = struct{}{}
			for _, quote := range quoteList { // <── loop on every quote
				days = append(days, day)
				bases = append(bases, base)
				quotes = append(quotes, strings.ToUpper(quote))
			}
		}
	}

	// UNNEST is wanted in the case of multiples quotes for one base. Avoiding return those quotes for every base.
	query := `
		WITH wanted AS (
		SELECT *
			FROM unnest($1::date[], $2::text[], $3::text[])
				AS t(day, base_currency, quote_currency)
		)
		SELECT f.day, f.base_currency, f.quote_currency, f.rate
		FROM fx_rates AS f
		JOIN wanted USING (day, base_currency, quote_currency);
		`

	rows, err := s.db.Query(query,
		pq.Array(days),
		pq.Array(bases),
		pq.Array(quotes))
	if err != nil {
		return nil, fmt.Errorf("failed to query rates: %v", err)
	}
	defer rows.Close()

	rates := make(Rates, len(days))
	for rows.Next() {
		var day time.Time
		var base, quote string
		var rate float64

		if err := rows.Scan(&day, &base, &quote, &rate); err != nil {
			return nil, fmt.Errorf("failed to query rates: %v", err)
		}

		dateStr := fx.FormatToString(day)
		if _, exists := rates[dateStr]; !exists {
			rates[dateStr] = make(map[string]map[string]float64, len(bases))
		}
		if _, exists := rates[dateStr][base]; !exists {
			rates[dateStr][base] = make(map[string]float64, len(quotes))
		}
		rates[dateStr][base][quote] = rate
		delete(notFound[dateStr], base)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query rates: %v", err)
	}

	// ── 3) background fetch for missing rates  ────────────────────────────
	for day, missingBases := range notFound {
		for base := range missingBases {
			dayStr, err := fx.FormatToTime(day)
			if err != nil {
				log.Printf("Error formating date string to time: %v", err)
				return rates, nil
			}
			go s.fetchRatesAndUpdateTable(base, dayStr)
		}
	}

	return rates, nil
}

func (s *databaseStore) fetchRatesAndUpdateTable(currency string, date time.Time) error {
	rates, err := fx.RatesOn(currency, date)
	if err != nil {
		return fmt.Errorf("failed to get rate: %v", err)
	}
	KeepOnlyValidCurrencies(rates, currencyCatalog)
	if err := s.bulkUpsertRates(date, currency, rates); err != nil {
		return fmt.Errorf("error during upsert rates: %v", err)
	}
	log.Printf("Updated rates for %s@%s\n", currency, fx.FormatToString(date))
	return nil
}

func (s *databaseStore) bulkUpsertRates(date time.Time, currency string, rates map[string]float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 1) temp table, removed at COMMIT/ROLLBACK
	if _, err := tx.Exec(`
        CREATE TEMP TABLE tmp_fx_rates
        (LIKE fx_rates INCLUDING DEFAULTS)
        ON COMMIT DROP`); err != nil {
		return fmt.Errorf("failed to create tmp_fx_rates: %v", err)
	}
	// 2) COPY In
	stmt, err := tx.Prepare(pq.CopyIn(
		"tmp_fx_rates",
		"day", "base_currency", "quote_currency", "rate",
	))
	if err != nil {
		return fmt.Errorf("failed to prepare copy in for rates: %v", err)
	}
	defer stmt.Close()

	for quote, rate := range rates {
		if _, err := stmt.Exec(date, currency, quote, rate); err != nil {
			return fmt.Errorf("failed to execute copy in for rates: %v", err)
		}
	}
	if _, err = stmt.Exec(); err != nil {
		return fmt.Errorf("failed to finalize copy in for rates: %v", err)
	}

	// 3) upsert rows from temp table
	res, err := tx.Exec(`
        INSERT INTO fx_rates (day, base_currency, quote_currency, rate)
        SELECT day, base_currency, quote_currency, rate
          FROM tmp_fx_rates
        ON CONFLICT (day, base_currency, quote_currency)
        DO NOTHING;`)
	if err != nil {
		return fmt.Errorf("failed to execute upsert from tmp_fx_rates for rates: %v", err)
	}
	if rows, _ := res.RowsAffected(); rows > 0 {
		log.Printf("fx_rates: inserted %d new rows", rows)
		log.Printf("Updated rates for %s@%s\n", currency, fx.FormatToString(date))
	}

	return tx.Commit()
}
