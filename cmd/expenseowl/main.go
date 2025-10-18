package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/tanq16/expenseowl/internal/api"
	"github.com/tanq16/expenseowl/internal/storage"
	"github.com/tanq16/expenseowl/internal/web"
)

var version = "dev"

func runServer(port int) {
	storage, err := storage.InitializeStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()
	handler := api.NewHandler(storage)

	// Version Handler
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(version))
	})

	// UI Handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		if err := web.ServeTemplate(w, "index.html"); err != nil {
			log.Printf("HTTP ERROR: Failed to serve template: %v", err)
			http.Error(w, "Failed to serve template", http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("/table", handler.ServeTableView)
	http.HandleFunc("/settings", handler.ServeSettingsPage)

	// Static File Handlers
	http.HandleFunc("/functions.js", handler.ServeStaticFile)
	http.HandleFunc("/manifest.json", handler.ServeStaticFile)
	http.HandleFunc("/worldCurrencies.json", handler.ServeStaticFile)
	http.HandleFunc("/sw.js", handler.ServeStaticFile)
	http.HandleFunc("/pwa/", handler.ServeStaticFile)
	http.HandleFunc("/style.css", handler.ServeStaticFile)
	http.HandleFunc("/favicon.ico", handler.ServeStaticFile)
	http.HandleFunc("/chart.min.js", handler.ServeStaticFile)
	http.HandleFunc("/fa.min.css", handler.ServeStaticFile)
	http.HandleFunc("/webfonts/", handler.ServeStaticFile)

	// Config
	http.HandleFunc("/config", handler.GetConfig)
	http.HandleFunc("/categories", handler.GetCategories)
	http.HandleFunc("/categories/edit", handler.UpdateCategories)
	http.HandleFunc("/currency", handler.GetDefaultCurrency)
	http.HandleFunc("/currency/edit", handler.UpdateDefaultCurrency)
	http.HandleFunc("/currency/catalog", handler.GetCurrencyCatalog)
	http.HandleFunc("/startdate", handler.GetStartDate)
	http.HandleFunc("/startdate/edit", handler.UpdateStartDate)
	// http.HandleFunc("/tags", handler.GetTags)
	// http.HandleFunc("/tags/edit", handler.UpdateTags)

	// Expenses
	http.HandleFunc("/expense", handler.AddExpense)                     // PUT for add
	http.HandleFunc("/expenses", handler.GetExpenses)                   // GET all
	http.HandleFunc("/expense/edit", handler.EditExpense)               // PUT for edit
	http.HandleFunc("/expense/delete", handler.DeleteExpense)           // DELETE for single
	http.HandleFunc("/expenses/delete", handler.DeleteMultipleExpenses) // DELETE for multiple

	// Recurring Expenses
	http.HandleFunc("/recurring-expense", handler.AddRecurringExpense)           // PUT for add
	http.HandleFunc("/recurring-expenses", handler.GetRecurringExpenses)         // GET all
	http.HandleFunc("/recurring-expense/edit", handler.UpdateRecurringExpense)   // PUT for edit
	http.HandleFunc("/recurring-expense/delete", handler.DeleteRecurringExpense) // DELETE

	// Fx
	http.HandleFunc("/fx/rate", handler.GetRate) // GET
	http.HandleFunc("/fx/rates", handler.GetRates) // POST

	// Import/Export
	http.HandleFunc("/export/csv", handler.ExportCSV)
	http.HandleFunc("/import/csv", handler.ImportCSV)
	http.HandleFunc("/import/csvold", handler.ImportOldCSV)

	log.Println("Starting server on port", port, "...")
	if err := http.ListenAndServe(fmt.Sprint(":", port), nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func main() {
	port := flag.Int("port", 8080, "Port to serve from")
	flag.Parse()
	runServer(*port)
}
