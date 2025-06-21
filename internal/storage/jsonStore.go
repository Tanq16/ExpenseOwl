package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
)

// no-ops for JSON storage since it's file based
func (s *jsonStore) Initialize() error {
	return nil
}
func (s *jsonStore) Close() error {
	return nil
}

func (s *jsonStore) SaveExpense(expense *Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readFile()
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	if expense.ID == "" {
		expense.ID = uuid.New().String()
	}
	if expense.Date.IsZero() {
		expense.Date = time.Now()
	}
	data.Expenses = append(data.Expenses, expense)
	log.Printf("Added expense with ID %s\n", expense.ID)
	return s.writeFile(data)
}

func (s *jsonStore) GetExpense(id string) (*Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.readFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read storage file: %v", err)
	}
	for i, exp := range data.Expenses {
		if exp.ID == id {
			return data.Expenses[i], nil
		}
	}
	return nil, fmt.Errorf("expense with ID %s not found", id)
}

func (s *jsonStore) DeleteExpense(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readFile()
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	found := false
	newExpenses := make([]*Expense, 0, len(data.Expenses)-1)
	for _, exp := range data.Expenses {
		if exp.ID != id {
			newExpenses = append(newExpenses, exp)
		} else {
			found = true
		}
	}
	// log.Printf("Looped to find expense with ID %s. Found: %v\n", id, found)
	if !found {
		return fmt.Errorf("expense with ID %s not found", id)
	}
	data.Expenses = newExpenses
	log.Printf("Deleted expense with ID %s\n", id)
	return s.writeFile(data)
}

func (s *jsonStore) EditExpense(expense *Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readFile()
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	found := false
	for i, exp := range data.Expenses {
		if exp.ID == expense.ID {
			expense.Date = exp.Date
			data.Expenses[i] = expense
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("expense with ID %s not found", expense.ID)
	}
	log.Printf("Edited expense with ID %s\n", expense.ID)
	return s.writeFile(data)
}

func (s *jsonStore) GetAllExpenses() ([]*Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.readFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read storage file: %v", err)
	}
	log.Println("Retrieved all expenses")
	return data.Expenses, nil
}

func (s *jsonStore) readFile() (*fileData, error) {
	content, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}
	var data fileData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *jsonStore) writeFile(data *fileData) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, content, 0644)
}
