package fx

import (
	"container/list"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Fx interface {
	RatesOn(base string, day time.Time) (map[string]float64, error)
	TodayRates(base string) (map[string]float64, error)
	Rate(from, to string, day time.Time) (float64, error)
	TodayRate(from, to string) (float64, error)
}

/* ----------------------------------------------------------------------- */
/* --------------------------- LRU cache --------------------------------- */
/* ----------------------------------------------------------------------- */

// cacheKey looks like "usd@2025-07-08"
type CacheKey string

// daySnapshot holds all FX pairs for one (base, date) combo.
type DaySnapshot struct {
	Key   CacheKey
	Rates map[string]float64
}

// LRUCache is a threadsafe, fixed-capacity LRU store.
// It keeps at most Capacity snapshots in memory.
type LRUCache struct {
	Capacity int
	Mu       sync.Mutex
	Order    *list.List                 // front = most-recent
	Entries  map[CacheKey]*list.Element // key → list node
}

// NewLRUCache returns an empty cache with the given capacity.
func NewLRUCache(cap int) *LRUCache {
	return &LRUCache{
		Capacity: cap,
		Order:    list.New(),
		Entries:  make(map[CacheKey]*list.Element, cap),
	}
}

// Get moves the requested snapshot to the front and returns a copy.
// ok == false when the key is absent.
func (c *LRUCache) Get(key CacheKey) (rates map[string]float64, ok bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if node, hit := c.Entries[key]; hit {
		c.Order.MoveToFront(node)
		// shallow copy: caller can mutate safely
		ratesCopy := make(map[string]float64, len(node.Value.(*DaySnapshot).Rates))
		for k, v := range node.Value.(*DaySnapshot).Rates {
			ratesCopy[k] = v
		}
		return ratesCopy, true
	}
	return nil, false
}

// Put inserts or updates a snapshot and evicts the LRU item if full.
func (c *LRUCache) Put(key CacheKey, rates map[string]float64) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if node, exists := c.Entries[key]; exists {
		node.Value.(*DaySnapshot).Rates = rates
		c.Order.MoveToFront(node)
		return
	}

	if c.Order.Len() >= c.Capacity {
		lru := c.Order.Back()
		c.Order.Remove(lru)
		delete(c.Entries, lru.Value.(*DaySnapshot).Key)
	}

	node := c.Order.PushFront(&DaySnapshot{Key: key, Rates: rates})
	c.Entries[key] = node
}

/* ------------------------------------------------------------------------- */
/* ----------------------------- core logic -------------------------------- */
/* ------------------------------------------------------------------------- */

var (
	httpClient  = &http.Client{Timeout: 5 * time.Second}
	cache       = NewLRUCache(20) // ≤20 day-snapshots in RAM
	isoLayout   = "2006-01-02"
	urlTemplate = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@%s/v1/currencies/%s.json"
)

/* ------------------------ low-level fetch -------------------------------- */

// pull one JSON blob (all FX pairs) for `base` on `day`
func fetchFromWeb(base string, day time.Time) (map[string]float64, error) {
	url := fmt.Sprintf(urlTemplate, day.Format(isoLayout), base)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fx: http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx: http %s", resp.Status)
	}

	// 1) unmarshal to RawMessage so we can pluck out the nested map
	var tmp map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&tmp); err != nil {
		return nil, fmt.Errorf("fx: json decode: %w", err)
	}

	// 2) sanity-check that the payload really contains the base key
	raw, ok := tmp[base]
	if !ok {
		return nil, fmt.Errorf("fx: key %q not found in payload", base)
	}

	// 3) finally unmarshal that nested object into the map we need
	rates := make(map[string]float64)
	if err := json.Unmarshal(raw, &rates); err != nil {
		return nil, fmt.Errorf("fx: nested decode: %w", err)
	}

	return rates, nil
}

/* ------------------------------------------------------------------------- */
/* -------------------------- public api ------------------------------- */
/* ------------------------------------------------------------------------- */
func FormatToString(date time.Time) string {
	return date.Format(isoLayout)
}

func FormatToTime(dateStr string) (time.Time, error) {
	return time.Parse(isoLayout, dateStr)
}

// RatesOn returns all rates for `base` (ISO, any case) on `day`.
func RatesOn(base string, day time.Time) (map[string]float64, error) {
	base = strings.ToLower(base)
	key := CacheKey(fmt.Sprintf("%s@%s", base, day.Format(isoLayout)))

	if snap, ok := cache.Get(key); ok {
		return snap, nil
	}

	snap, err := fetchFromWeb(base, day)
	if err != nil {
		return nil, err
	}
	cache.Put(key, snap)
	return snap, nil
}

// TodayRates wraps RatesOn using the current UTC date.
func TodayRates(base string) (map[string]float64, error) {
	return RatesOn(base, time.Now().UTC())
}

// Rate returns the rate `from` → `to` on `day` (or today if zero value).
func Rate(from, to string, day time.Time) (float64, error) {
	from = strings.ToLower(from)
	to = strings.ToLower(to)

	if from == to {
		return 1, nil
	}
	if day.IsZero() {
		day = time.Now().UTC()
	}
	rates, err := RatesOn(from, day)
	if err != nil {
		return 0, err
	}
	val, ok := rates[to]
	if !ok {
		return 0, fmt.Errorf("fx: rate %s→%s not present", from, to)
	}
	return val, nil
}

// TodayRates wraps Rate using the current UTC date.
func TodayRate(from, to string) (float64, error) {
	return Rate(from, to, time.Now().UTC())
}
