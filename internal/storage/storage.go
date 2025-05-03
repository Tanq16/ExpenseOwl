package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tanq16/expenseowl/internal/config"
)

var (
	ErrExpenseNotFound = errors.New("expense not found")
	ErrInvalidExpense  = errors.New("invalid expense data")
)

type Storage interface {
	SaveExpense(expense *config.Expense) error
	GetAllExpenses() ([]*config.Expense, error)
	DeleteExpense(id string) error
	EditExpense(expense *config.Expense) error
}

type jsonStore struct {
	filePath string
	mu       sync.RWMutex
}

type fileData struct {
	Expenses []*config.Expense `json:"expenses"`
}

func New(filePath string) (*jsonStore, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		initialData := fileData{Expenses: []*config.Expense{}}
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
		filePath: filePath,
	}, nil
}

func (s *jsonStore) SaveExpense(expense *config.Expense) error {
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

func (s *jsonStore) DeleteExpense(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readFile()
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	found := false
	newExpenses := make([]*config.Expense, 0, len(data.Expenses)-1)
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

func (s *jsonStore) EditExpense(expense *config.Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.readFile()
	if err != nil {
		return fmt.Errorf("failed to read storage file: %v", err)
	}
	found := false
	for i, exp := range data.Expenses {
		if exp.ID == expense.ID {
			data.Expenses[i] = expense
			found = true
			break
		}
	}
	if !found {
		return ErrExpenseNotFound
	}
	log.Printf("Edited expense with ID %s\n", expense.ID)
	return s.writeFile(data)
}

func (s *jsonStore) GetAllExpenses() ([]*config.Expense, error) {
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
