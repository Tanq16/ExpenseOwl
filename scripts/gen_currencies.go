//go:build ignore
// +build ignore

// Generates internal/config/currencies_gen.go from the local
// internal/web/templates/worldCurrencies.json (no network required).

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
)

const (
	srcFile = "internal/web/templates/worldCurrencies.json"
	outFile = "internal/storage/currencies.go"
)

// Slim is the structure of each entry in currencies_simplified.json
type Currency struct {
	Code            string `json:"code"`
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	Decimals        uint8  `json:"decimals"`
	CommaDecimal    bool   `json:"commaDecimal"`
	PostFixCurrency bool   `json:"postFixCurrency"`
}

// -------------------------------------------------------------------
// Template for the generated Go source
// -------------------------------------------------------------------
var genTmpl = template.Must(template.New("curr").
	Funcs(template.FuncMap{"lower": strings.ToLower}).
	Parse(
`package storage

import "strings"

func IsValidCurrency(code string) bool {
	_, ok := currencyCatalog[strings.ToUpper(code)]
	return ok
}

// keepOnly keeps in m only the keys that appear in the allow‐set.
//
//	m        : map you want to trim         (map[string]float64)
//	allowSet : membership test in O(1)      (map[string]string{})
func KeepOnlyValidCurrencies(m map[string]float64, allowSet map[string]string) {
	for k, v := range m {
		upperCase := strings.ToUpper(k)
		_, ok := allowSet[upperCase]
		delete(m, k)
		if ok {
			m[upperCase] = v
		}
	}
}

// CurrencyCatalog maps ISO-4217 codes to their symbol.
// The comment preserves the human-readable currency name.
var currencyCatalog = map[string]string{
{{- range . }}
	"{{ .Code }}": "{{ .Symbol }}", // {{ .Name }}
{{- end }}
}

`))

// -------------------------------------------------------------------

func main() {
	raw, err := os.ReadFile(srcFile)
	check(err)

	var currencies []Currency
	check(json.Unmarshal(raw, &currencies))

	// Ensure consistent ordering
	sort.Slice(currencies, func(i, j int) bool { return currencies[i].Code < currencies[j].Code })

	var buf bytes.Buffer
	check(genTmpl.Execute(&buf, currencies))

	check(os.WriteFile(outFile, buf.Bytes(), 0o644))
	fmt.Printf("🎉  Generated %d currencies → %s\n", len(currencies), outFile)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
