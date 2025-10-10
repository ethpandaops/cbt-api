package integrationtest

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Reporter formats and displays test results.
type Reporter struct {
	Writer io.Writer // Where to write output (os.Stdout or test log)
}

// NewReporter creates a new reporter.
func NewReporter(w io.Writer) *Reporter {
	return &Reporter{Writer: w}
}

// Report displays test results summary and detailed failure information.
func (r *Reporter) Report(results []TestResult) {
	// Calculate totals
	total := len(results)
	successes := 0
	failures := 0

	var totalDuration time.Duration

	for _, result := range results {
		totalDuration += result.Duration
		if result.Success {
			successes++
		} else {
			failures++
		}
	}

	// Calculate percentages
	successPct := 0.0
	failurePct := 0.0

	if total > 0 {
		successPct = float64(successes) / float64(total) * 100
		failurePct = float64(failures) / float64(total) * 100
	}

	// Print summary header
	fmt.Fprintf(r.Writer, "\nIntegration Test Results\n")
	fmt.Fprintf(r.Writer, "%s\n", strings.Repeat("=", 24))
	fmt.Fprintf(r.Writer, "Total Endpoints: %d\n", total)
	fmt.Fprintf(r.Writer, "Passed: %d (%.1f%%)\n", successes, successPct)
	fmt.Fprintf(r.Writer, "Failed: %d (%.1f%%)\n", failures, failurePct)
	fmt.Fprintf(r.Writer, "Duration: %s\n", totalDuration.Round(time.Millisecond))

	// Print failure details if any exist
	if failures > 0 {
		r.ReportFailures(results)
	} else {
		fmt.Fprintf(r.Writer, "\n\u2705 All endpoints returned 200 OK\n")
	}
}

// ReportSummary prints a one-line summary.
func (r *Reporter) ReportSummary(results []TestResult) {
	total := len(results)
	successes := 0

	for _, result := range results {
		if result.Success {
			successes++
		}
	}

	failures := total - successes
	successPct := 0.0

	if total > 0 {
		successPct = float64(successes) / float64(total) * 100
	}

	if failures == 0 {
		fmt.Fprintf(r.Writer, "%d/%d endpoints passed (%.1f%%)\n", successes, total, successPct)
	} else {
		fmt.Fprintf(r.Writer, "%d/%d endpoints passed (%.1f%%), %d failed\n", successes, total, successPct, failures)
	}
}

// ReportFailures prints detailed failure information.
func (r *Reporter) ReportFailures(results []TestResult) {
	failures := filterFailures(results)
	if len(failures) == 0 {
		fmt.Fprintf(r.Writer, "\n\u2705 All endpoints returned 200 OK\n")

		return
	}

	fmt.Fprintf(r.Writer, "\n\u274c Failed Endpoints:\n")
	fmt.Fprintf(r.Writer, "%s\n", strings.Repeat("-", 80))

	for _, result := range failures {
		fmt.Fprintf(r.Writer, "\nEndpoint: %s\n", result.Endpoint)
		fmt.Fprintf(r.Writer, "Status Code: %d\n", result.StatusCode)
		fmt.Fprintf(r.Writer, "Duration: %s\n", result.Duration.Round(time.Millisecond))
		fmt.Fprintf(r.Writer, "Error Body:\n%s\n", truncate(result.ErrorBody, 500))
		fmt.Fprintf(r.Writer, "%s\n", strings.Repeat("-", 80))
	}
}

// filterFailures returns only failed test results.
func filterFailures(results []TestResult) []TestResult {
	failures := make([]TestResult, 0, len(results))
	for _, result := range results {
		if !result.Success {
			failures = append(failures, result)
		}
	}

	return failures
}

// truncate limits string length with "..." suffix.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "..."
}
