package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type LogLine struct {
	Timestamp  string         `json:"timestamp"`
	Service    string         `json:"service"`
	Status     string         `json:"status"`
	Message    string         `json:"message"`
	Attributes map[string]any `json:"attributes"`
}

func main() {
	var (
		output = flag.String("output", "testdata/logs.jsonl", "output JSON Lines file path")
		count  = flag.Int("count", 1000, "number of log lines to generate")
		seed   = flag.Int64("seed", 1, "random seed")
	)

	flag.Parse()

	if err := run(*output, *count, *seed); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate testdata: %v\n", err)
		os.Exit(1)
	}
}

func run(output string, count int, seed int64) error {
	if count <= 0 {
		return fmt.Errorf("count must be positive")
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}

	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	r := rand.New(rand.NewSource(seed))
	encoder := json.NewEncoder(file)

	baseTime := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	for i := 0; i < count; i++ {
		record := generateLogLine(r, i, baseTime)

		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	return nil
}

func generateLogLine(r *rand.Rand, index int, baseTime time.Time) LogLine {
	services := []string{"api", "web", "worker", "batch", "auth"}

	service := services[r.Intn(len(services))]
	status := weightedStatus(r)

	timestamp := baseTime.Add(time.Duration(index) * time.Second)

	httpStatus := weightedHTTPStatus(r, status)

	return LogLine{
		Timestamp: timestamp.Format(time.RFC3339),
		Service:   service,
		Status:    status,
		Message:   generateMessage(r, service, status, httpStatus),
		Attributes: map[string]any{
			"request_id": fmt.Sprintf("req_%06d", index+1),
			"trace_id":   fmt.Sprintf("trace_%06d", r.Intn(100000)),
			"span_id":    fmt.Sprintf("span_%06d", r.Intn(100000)),
			"env":        choose(r, []string{"local", "dev", "staging", "production"}),

			"http": map[string]any{
				"method":      choose(r, []string{"GET", "POST", "PUT", "PATCH", "DELETE"}),
				"path":        choose(r, []string{"/api/users", "/api/projects", "/api/login", "/api/orders", "/health"}),
				"status_code": httpStatus,
				"duration_ms": r.Intn(3000) + 1,
			},

			"user": map[string]any{
				"id":   r.Intn(50) + 1,
				"name": choose(r, []string{"alice", "bob", "carol", "dave", "eve"}),
			},

			"host": map[string]any{
				"name": fmt.Sprintf("app-%02d", r.Intn(5)+1),
				"az":   choose(r, []string{"ap-northeast-1a", "ap-northeast-1c", "ap-northeast-1d"}),
			},

			"db": map[string]any{
				"statement":   choose(r, []string{"SELECT", "INSERT", "UPDATE", "DELETE"}),
				"duration_ms": r.Intn(1000) + 1,
			},
		},
	}
}

func weightedStatus(r *rand.Rand) string {
	n := r.Intn(100)

	switch {
	case n < 65:
		return "info"
	case n < 80:
		return "debug"
	case n < 92:
		return "warn"
	default:
		return "error"
	}
}

func weightedHTTPStatus(r *rand.Rand, status string) int {
	switch status {
	case "error":
		return chooseInt(r, []int{500, 502, 503, 504})
	case "warn":
		return chooseInt(r, []int{400, 401, 403, 404, 429})
	default:
		return chooseInt(r, []int{200, 201, 202, 204, 304})
	}
}

func generateMessage(r *rand.Rand, service string, status string, httpStatus int) string {
	if status == "error" {
		return choose(r, []string{
			"failed to connect database",
			"request timeout",
			"unexpected internal server error",
			"failed to publish event",
			"dependency service unavailable",
		})
	}

	if status == "warn" {
		return choose(r, []string{
			"slow request detected",
			"retrying job",
			"rate limit exceeded",
			"invalid request parameter",
			"cache miss rate is high",
		})
	}

	return fmt.Sprintf("%s handled request with status %d", service, httpStatus)
}

func choose(r *rand.Rand, values []string) string {
	return values[r.Intn(len(values))]
}

func chooseInt(r *rand.Rand, values []int) int {
	return values[r.Intn(len(values))]
}
