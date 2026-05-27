package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	baseURL = "http://localhost:8080"
	file    = "data/webhook_payloads.json"
	dsn     = "mysql:mysql@tcp(localhost:3306)/db?parseTime=true&loc=UTC"
)

type payload struct {
	Gateway string            `json:"gateway"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

func main() {
	raw, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("read file: %v", err)
	}

	var payloads []payload
	if err := json.Unmarshal(raw, &payloads); err != nil {
		log.Fatalf("parse json: %v", err)
	}

	fmt.Printf("sending %d payloads to %s\n\n", len(payloads), baseURL)

	counts := make(map[string]int)
	for i, p := range payloads {
		counts[send(p)]++
		if (i+1)%25 == 0 {
			fmt.Printf("  ... %d/%d\n", i+1, len(payloads))
		}
	}

	fmt.Print("\nwaiting for workers to drain... ")
	time.Sleep(3 * time.Second)
	fmt.Println("done")

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	leadEvents := queryCount(db, "SELECT COUNT(*) FROM lead_events WHERE event='order.approved'")
	decryptDLQ := queryCount(db, "SELECT COUNT(*) FROM lead_dead_letter WHERE origin='decrypt'")
	schemaDLQ := queryCount(db, "SELECT COUNT(*) FROM lead_dead_letter WHERE origin='schema_validation'")

	apiDupes := counts["duplicate"]
	dbDupes := max(counts["accepted"] - leadEvents, 0)

	fmt.Printf("\n=== HTTP responses ===\n")
	for k, v := range counts {
		fmt.Printf("  %-15s %d\n", k, v)
	}

	fmt.Printf("\n=== Resultado canônico (banco) ===\n")
	fmt.Printf("  valid_approved_unique_in_lead_events  %d  (esperado 10)\n", leadEvents)
	fmt.Printf("  dlq_decrypt_failed                    %d  (esperado 1)\n", decryptDLQ)
	fmt.Printf("  dlq_schema_failed                     %d  (esperado 3)\n", schemaDLQ)
	fmt.Printf("  duplicates_natural_key                %d  (esperado 4)  [API:%d  DB:%d]\n",
		apiDupes+dbDupes, apiDupes, dbDupes)
	fmt.Printf("  non_approved_discarded                %d  (esperado 2)\n", counts["discarded"])
}

func send(p payload) string {
	req, err := http.NewRequest(http.MethodPost, baseURL+"/webhooks/"+p.Gateway, bytes.NewReader(p.Body))
	if err != nil {
		return "build_error"
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "send_error"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Status string `json:"status"`
	}
	json.Unmarshal(body, &r)
	if r.Status != "" {
		return r.Status
	}
	return fmt.Sprintf("http_%d", resp.StatusCode)
}

func queryCount(db *sql.DB, query string) int {
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		log.Printf("query failed: %v", err)
		return -1
	}
	return n
}
