package api

// func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		log.Println("HTTP ERROR: Method not allowed")
// 		return
// 	}
// 	expenses, err := h.storage.GetAllExpenses()
// 	if err != nil {
// 		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve expenses"})
// 		log.Printf("HTTP ERROR: Failed to retrieve expenses: %v\n", err)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "text/csv")
// 	w.Header().Set("Content-Disposition", "attachment; filename=expenses.csv")
// 	// write CSV data
// 	w.Write([]byte("ID,Name,Category,Amount,Date\n"))
// 	for _, expense := range expenses {
// 		line := fmt.Sprintf("%s,%s,%s,%.2f,%s\n",
// 			expense.ID,
// 			strings.ReplaceAll(expense.Name, ",", ";"), // Replace , in name with ;
// 			expense.Category,
// 			expense.Amount,
// 			expense.Date.Format("2006-01-02 15:04:05"),
// 		)
// 		w.Write([]byte(line))
// 	}
// 	log.Println("HTTP: Exported expenses to CSV")
// }

// func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		log.Println("HTTP ERROR: Method not allowed")
// 		return
// 	}
// 	err := r.ParseMultipartForm(10 << 20) // 10MB max
// 	if err != nil {
// 		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Error parsing form"})
// 		log.Printf("HTTP ERROR: Error parsing multipart form: %v\n", err)
// 		return
// 	}
// 	file, _, err := r.FormFile("file")
// 	if err != nil {
// 		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Error retrieving file"})
// 		log.Printf("HTTP ERROR: Error retrieving file from form: %v\n", err)
// 		return
// 	}
// 	defer file.Close()

// 	reader := csv.NewReader(file)
// 	records, err := reader.ReadAll()
// 	if err != nil {
// 		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Error reading CSV file"})
// 		log.Printf("HTTP ERROR: Error reading CSV file: %v\n", err)
// 		return
// 	}
// 	if len(records) < 2 { // header + at least one data row
// 		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "CSV file has no data rows"})
// 		log.Println("HTTP ERROR: CSV file is empty or has no data rows")
// 		return
// 	}
// 	stringEscape := regexp.MustCompile(`[^a-zA-Z0-9_ \.]*`)
// 	header := records[0]
// 	// Find the indices of required columns
// 	var nameIdx, categoryIdx, amountIdx, dateIdx int = -1, -1, -1, -1
// 	for i, col := range header {
// 		colLower := strings.ToLower(strings.TrimSpace(stringEscape.ReplaceAllString(col, "")))
// 		switch colLower {
// 		case "name":
// 			nameIdx = i
// 		case "category":
// 			categoryIdx = i
// 		case "amount":
// 			amountIdx = i
// 		case "date":
// 			dateIdx = i
// 		}
// 	}
// 	if nameIdx == -1 || categoryIdx == -1 || amountIdx == -1 || dateIdx == -1 {
// 		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "CSV missing required columns"})
// 		log.Println("HTTP ERROR: CSV file missing required columns")
// 		return
// 	}

// 	// Get current categories as lowercase to match new ones and replace as needed
// 	categoryMap := make(map[string]string)
// 	for _, cat := range h.config.Categories {
// 		catLower := strings.ToLower(cat)
// 		categoryMap[catLower] = cat
// 	}
// 	// Process data rows
// 	imported := 0
// 	var newCategories []string
// 	for i, record := range records {
// 		if i == 0 { // Skip header
// 			continue
// 		}
// 		// Considering max to support any CSV as long as it has the required columns
// 		if len(record) <= slices.Max([]int{nameIdx, categoryIdx, amountIdx, dateIdx}) {
// 			log.Printf("Warning: Skipping row %d due to insufficient columns\n", i)
// 			continue
// 		}
// 		// Handle name
// 		name := strings.TrimSpace(stringEscape.ReplaceAllString(record[nameIdx], ""))
// 		if name == "" {
// 			name = "-"
// 		}
// 		// Handle category
// 		rawCategory := strings.TrimSpace(stringEscape.ReplaceAllString(record[categoryIdx], ""))
// 		if rawCategory == "" {
// 			log.Printf("Warning: Skipping row %d due to missing category\n", i)
// 			continue
// 		}
// 		categoryLower := strings.ToLower(rawCategory)
// 		category := rawCategory
// 		if normalized, exists := categoryMap[categoryLower]; exists { // Matching lowercase category
// 			category = normalized
// 		} else { // New category found
// 			categoryMap[categoryLower] = rawCategory // Add to map for future steps
// 			newCategories = append(newCategories, rawCategory)
// 		}
// 		// Handle amount (skipping regex since parsing as float)
// 		amount, err := strconv.ParseFloat(strings.TrimSpace(record[amountIdx]), 64)
// 		if err != nil || amount <= 0 {
// 			log.Printf("Warning: Skipping row %d due to invalid amount: %s\n", i, record[amountIdx])
// 			continue
// 		}
// 		// Handle date (skipping regex since parsing as time)
// 		dateStr := strings.TrimSpace(record[dateIdx])
// 		var date time.Time
// 		var parsedDate bool
// 		dateFormats := []string{ // Common date formats
// 			time.RFC3339,          // 2006-01-02T15:04:05Z07:00
// 			"2006-01-02 15:04:05", // SQL format
// 			"2006-01-02",          // ISO date
// 			"01/02/2006",          // US date
// 			"01/02/06",            // US date (short year)
// 			"1/2/2006",            // US date (no leading zero)
// 			"1/2/06",              // US date (short year no leading zero)
// 			"Jan 2, 2006",         // Month name format
// 			"January 2, 2006",     // Full month name
// 			"2006-01-02T15:04:05", // ISO without timezone
// 		}
// 		for _, format := range dateFormats {
// 			if d, err := time.ParseInLocation(format, dateStr, time.Local); err == nil {
// 				date = d.UTC()
// 				parsedDate = true
// 				break
// 			}
// 		}
// 		if !parsedDate {
// 			log.Printf("Warning: Skipping row %d due to invalid date format: %s\n", i, dateStr)
// 			continue
// 		}
// 		// Save the expense
// 		expense := &storage.Expense{
// 			ID:       "", // Ensure new ID value
// 			Name:     name,
// 			Category: category,
// 			Amount:   amount,
// 			Date:     date,
// 		}

// 		if err := h.storage.AddExpense(expense); err != nil {
// 			log.Printf("Error saving expense from row %d: %v\n", i, err)
// 			continue
// 		}
// 		imported++
// 		// Throttle for storage - usually not needed but can avoid error for large files
// 		time.Sleep(10 * time.Millisecond)
// 	}

// 	// Update the config with new categories if any
// 	if len(newCategories) > 0 {
// 		updatedCategories := append(h.config.Categories, newCategories...)
// 		if err := h.storage.UpdateCategories(updatedCategories); err != nil {
// 			log.Printf("Warning: Failed to update categories: %v\n", err)
// 		}
// 	}
// 	// Return success response with summary
// 	writeJSON(w, http.StatusOK, map[string]any{
// 		"status":          "success",
// 		"imported":        imported,
// 		"new_categories":  newCategories,
// 		"skipped":         len(records) - 1 - imported,
// 		"total_processed": len(records) - 1,
// 	})
// 	log.Printf("HTTP: Imported %d expenses from CSV file\n", imported)
// }
