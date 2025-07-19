package api

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tanq16/expenseowl/internal/storage"
)

// exports all expenses to CSV
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}
	expenses, err := h.storage.GetAllExpenses()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve expenses"})
		log.Printf("API ERROR: Failed to retrieve expenses for CSV export: %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=expenses.csv")
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	headers := []string{"ID", "Name", "Category", "Amount", "Date", "Tags"}
	if err := writer.Write(headers); err != nil {
		log.Printf("API ERROR: Failed to write CSV header: %v\n", err)
		return
	}

	// Write records
	for _, expense := range expenses {
		record := []string{
			expense.ID,
			expense.Name,
			expense.Category,
			expense.Currency,
			strconv.FormatFloat(expense.Amount, 'f', 2, 64),
			expense.Date.Format(time.RFC3339),
			strings.Join(expense.Tags, ","),
		}
		if err := writer.Write(record); err != nil {
			log.Printf("API ERROR: Failed to write CSV record for expense ID %s: %v\n", expense.ID, err)
			continue
		}
	}
	log.Println("HTTP: Exported expenses to CSV")
}

// imports expenses from CSV
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max file size
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Could not parse multipart form"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Error retrieving the file"})
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Failed to read CSV file"})
		return
	}
	if len(records) < 2 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "CSV file must have a header and at least one data row"})
		return
	}

	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}
	// Check for mandatory columns
	requiredCols := []string{"name", "category", "amount", "date"}
	for _, col := range requiredCols {
		if _, ok := colMap[col]; !ok {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("Missing required column: %s", col)})
			return
		}
	}
	// Get optional column indices
	idIdx, idExists := colMap["id"]
	tagsIdx, tagsExists := colMap["tags"]
	currencyIdx, currencyExists := colMap["currency"]

	currentCategories, err := h.storage.GetCategories()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Could not retrieve current categories"})
		return
	}
	categorySet := make(map[string]bool)
	for _, cat := range currentCategories {
		categorySet[strings.ToLower(cat)] = true
	}
	var newCategories []string
	var importedCount, skippedCount int
	// TODO: might be worth setting default currency when we have currency updation behavior
	currencyVal, err := h.storage.GetCurrency()
	if err != nil {
		log.Printf("Error: Could not retrieve currency, shutting down import: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Could not retrieve currency"})
		return
	}

	for i, record := range records[1:] {
		if len(record) != len(header) {
			log.Printf("Warning: Skipping row %d due to incorrect column count\n", i+2)
			skippedCount++
			continue
		}

		// Check if expense exists by ID, if provided - without doing a clash resolution
		if idExists {
			id := record[idIdx]
			if _, err := h.storage.GetExpense(id); err == nil {
				log.Printf("Info: Skipping row %d because expense with ID '%s' already exists\n", i+2, id)
				skippedCount++
				continue
			}
		}

		// Check for currency field, if provided - default is retrieved
		localCurrency := currencyVal
		if currencyExists {
			currency := record[currencyIdx]
			if !slices.Contains(storage.SupportedCurrencies, currency) {
				log.Printf("Warning: Skipping row %d due to invalid currency: %s\n", i+2, currency)
				skippedCount++
				continue
			}
			localCurrency = strings.TrimSpace(currency)
		}

		amount, err := strconv.ParseFloat(record[colMap["amount"]], 64)
		if err != nil {
			log.Printf("Warning: Skipping row %d due to invalid amount: %s\n", i+2, record[colMap["amount"]])
			skippedCount++
			continue
		}
		date, err := parseDate(record[colMap["date"]])
		if err != nil {
			log.Printf("Warning: Skipping row %d due to invalid date: %v\n", i+2, err)
			skippedCount++
			continue
		}
		category := strings.TrimSpace(record[colMap["category"]])
		if _, ok := categorySet[strings.ToLower(category)]; !ok {
			newCategories = append(newCategories, category)
			categorySet[strings.ToLower(category)] = true // Add to set to handle duplicates in the same file
		}
		var tags []string
		if tagsExists {
			tagsStr := record[tagsIdx]
			if tagsStr != "" {
				tags = strings.Split(tagsStr, ",")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
			}
		}

		expense := storage.Expense{
			Name:     strings.TrimSpace(record[colMap["name"]]),
			Category: category,
			Amount:   amount,
			Currency: localCurrency,
			Date:     date,
			Tags:     tags,
		}
		if err := expense.Validate(); err != nil {
			log.Printf("Warning: Skipping row %d due to validation error: %v\n", i+2, err)
			skippedCount++
			continue
		}
		if err := h.storage.AddExpense(expense); err != nil {
			log.Printf("Error: Could not add expense from row %d: %v\n", i+2, err)
			skippedCount++
			continue
		}
		importedCount++
		time.Sleep(10 * time.Millisecond) // Throttle to reduce storage overhead
	}

	if len(newCategories) > 0 {
		if err := h.storage.UpdateCategories(append(currentCategories, newCategories...)); err != nil {
			log.Printf("Warning: Failed to add new categories to config: %v\n", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "success",
		"total_processed": len(records) - 1,
		"imported":        importedCount,
		"skipped":         skippedCount,
		"new_categories":  newCategories,
	})
	log.Printf("HTTP: Imported %d expenses from CSV file. Skipped %d records.", importedCount, skippedCount)
}

func parseDate(dateStr string) (time.Time, error) {
	dateFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"1/2/2006",
		"01/02/2006",
		"1-2-2006",
		"01-02-2006",
	}
	for _, format := range dateFormats {
		if d, err := time.Parse(format, dateStr); err == nil {
			return d.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
