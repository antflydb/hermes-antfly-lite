package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/antflydb/hermes-antfly/internal/buildinfo"
	"github.com/antflydb/hermes-antfly/internal/evidence"
	"github.com/antflydb/hermes-antfly/internal/lite"
)

type evalCase struct {
	Name               string   `json:"name"`
	Query              string   `json:"query"`
	ExpectedIDs        []string `json:"expected_ids"`
	ForbiddenIDs       []string `json:"forbidden_ids"`
	RequiredRiskLabels []string `json:"required_risk_labels"`
	MaxHits            *int     `json:"max_hits,omitempty"`
}

type caseResult struct {
	Name           string   `json:"name"`
	Passed         bool     `json:"passed"`
	ReturnedIDs    []string `json:"returned_ids"`
	MissingIDs     []string `json:"missing_ids,omitempty"`
	ForbiddenIDs   []string `json:"forbidden_ids,omitempty"`
	MissingLabels  []string `json:"missing_risk_labels,omitempty"`
	InvalidSources []string `json:"invalid_sources,omitempty"`
	LatencyMS      float64  `json:"latency_ms"`
}

type report struct {
	Passed           bool         `json:"passed"`
	Build            string       `json:"build"`
	Cases            int          `json:"cases"`
	CasesPassed      int          `json:"cases_passed"`
	ExpectedRecall   float64      `json:"expected_recall"`
	ForbiddenLeakage int          `json:"forbidden_leakage"`
	CitationValidity float64      `json:"citation_validity"`
	OpenMS           float64      `json:"open_ms"`
	QueryP50MS       float64      `json:"query_p50_ms"`
	QueryP95MS       float64      `json:"query_p95_ms"`
	Results          []caseResult `json:"results"`
}

func main() {
	dbPath := flag.String("db", "", "existing .aflite knowledge database")
	suitePath := flag.String("suite", "", "JSONL retrieval evaluation suite")
	limit := flag.Int("limit", 6, "maximum results per query")
	showVersion := flag.Bool("version", false, "print build identity and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}
	if *dbPath == "" || *suitePath == "" || *limit < 1 || *limit > 20 {
		fmt.Fprintln(os.Stderr, "error: --db, --suite, and --limit between 1 and 20 are required")
		os.Exit(2)
	}

	cases, err := loadCases(*suitePath)
	if err != nil {
		fail("load evaluation suite", err)
	}
	openStarted := time.Now()
	store, err := lite.OpenReadonly(*dbPath)
	if err != nil {
		fail("open knowledge database", err)
	}
	defer store.Close()

	result := report{Build: buildinfo.String(), Cases: len(cases), OpenMS: milliseconds(time.Since(openStarted)), Results: make([]caseResult, 0, len(cases))}
	totalExpected, foundExpected, totalCitations, validCitations := 0, 0, 0, 0
	latencies := make([]float64, 0, len(cases))
	for _, testCase := range cases {
		queryStarted := time.Now()
		raw, err := store.Search(context.Background(), testCase.Query, *limit)
		latency := milliseconds(time.Since(queryStarted))
		if err != nil {
			fail("search evaluation case "+testCase.Name, err)
		}
		var search evidence.SearchResult
		if err := json.Unmarshal(raw, &search); err != nil {
			fail("decode evaluation result "+testCase.Name, err)
		}
		caseReport := evaluateCase(testCase, search)
		caseReport.LatencyMS = latency
		latencies = append(latencies, latency)
		for _, expected := range testCase.ExpectedIDs {
			totalExpected++
			if contains(caseReport.ReturnedIDs, expected) {
				foundExpected++
			}
		}
		for _, hit := range search.Hits {
			totalCitations++
			if hit.Citation.URL != "" && hit.Citation.Title != "" && hit.Citation.UpdatedAt != "" {
				validCitations++
			}
		}
		if caseReport.Passed {
			result.CasesPassed++
		}
		result.ForbiddenLeakage += len(caseReport.ForbiddenIDs)
		result.Results = append(result.Results, caseReport)
	}
	if totalExpected == 0 {
		result.ExpectedRecall = 1
	} else {
		result.ExpectedRecall = float64(foundExpected) / float64(totalExpected)
	}
	if totalCitations == 0 {
		result.CitationValidity = 1
	} else {
		result.CitationValidity = float64(validCitations) / float64(totalCitations)
	}
	result.Passed = result.CasesPassed == result.Cases && result.ExpectedRecall == 1 && result.ForbiddenLeakage == 0 && result.CitationValidity == 1
	sort.Float64s(latencies)
	result.QueryP50MS = percentile(latencies, 0.50)
	result.QueryP95MS = percentile(latencies, 0.95)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fail("encode evaluation report", err)
	}
	if !result.Passed {
		os.Exit(1)
	}
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}

func percentile(sorted []float64, quantile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * quantile)
	return sorted[index]
}

func loadCases(path string) ([]evalCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var cases []evalCase
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		var testCase evalCase
		if err := json.Unmarshal(scanner.Bytes(), &testCase); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if testCase.Name == "" || testCase.Query == "" {
			return nil, fmt.Errorf("line %d: name and query are required", line)
		}
		cases = append(cases, testCase)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("suite is empty")
	}
	return cases, nil
}

func evaluateCase(testCase evalCase, search evidence.SearchResult) caseResult {
	result := caseResult{Name: testCase.Name, ReturnedIDs: make([]string, 0, len(search.Hits))}
	labels := map[string]bool{}
	for _, hit := range search.Hits {
		result.ReturnedIDs = append(result.ReturnedIDs, hit.ID)
		for _, label := range hit.RiskLabels {
			labels[label] = true
		}
		if hit.Citation.URL == "" || hit.Citation.Title == "" || hit.Citation.UpdatedAt == "" {
			result.InvalidSources = append(result.InvalidSources, hit.ID)
		}
	}
	for _, expected := range testCase.ExpectedIDs {
		if !contains(result.ReturnedIDs, expected) {
			result.MissingIDs = append(result.MissingIDs, expected)
		}
	}
	for _, forbidden := range testCase.ForbiddenIDs {
		if contains(result.ReturnedIDs, forbidden) {
			result.ForbiddenIDs = append(result.ForbiddenIDs, forbidden)
		}
	}
	for _, label := range testCase.RequiredRiskLabels {
		if !labels[label] {
			result.MissingLabels = append(result.MissingLabels, label)
		}
	}
	tooMany := testCase.MaxHits != nil && len(search.Hits) > *testCase.MaxHits
	result.Passed = len(result.MissingIDs) == 0 && len(result.ForbiddenIDs) == 0 && len(result.MissingLabels) == 0 && len(result.InvalidSources) == 0 && !tooMany
	sort.Strings(result.ReturnedIDs)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", operation, err)
	os.Exit(1)
}
