// Command benchreport parses Prometheus metrics snapshots and optionally
// go-test-json output to produce a human-readable benchmark report.
//
// Usage:
//
//	benchreport [--impl=cached] --metrics-file=snap.txt \
//	            [--impl=direct] --metrics-file=snap-direct.txt \
//	            [--test-json=test.json] --out=report/REPORT.md
//
// Each --metrics-file must be preceded by --impl=<label> to tag that snapshot.
// Multiple snapshots (one per impl) are merged into the same report table.
// If --impl is omitted, the label defaults to "default".
//
// Metric names parsed:
//
//	ecp_gateway_auth_middleware_duration_seconds{provider="…"}
//	ecp_gateway_authz_check_duration_seconds{impl="…"}
//	ecp_gateway_rbac_fetch_duration_seconds{impl="…"}
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// histStats holds computed statistics for one histogram series.
type histStats struct {
	impl  string
	count uint64
	sum   float64
	p50   float64
	p90   float64
	p99   float64
}

func (h histStats) avg() float64 {
	if h.count == 0 {
		return 0
	}
	return h.sum / float64(h.count)
}

// snapshot holds a parsed metrics snapshot tagged with an impl label.
type snapshot struct {
	impl     string
	families map[string]*dto.MetricFamily
}

func main() {
	var (
		out      = flag.String("out", "report/REPORT.md", "output markdown file")
		testJSON = flag.String("test-json", "", "go test -json output file (optional)")
	)
	flag.Parse()

	// Collect (impl, file/url) pairs from the remaining args.
	// Supported forms: --impl=cached --metrics-file=snap.txt
	// or interleaved on the command line.
	snaps, err := parseSnapshotArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchreport: %v\n", err)
		os.Exit(1)
	}
	if len(snaps) == 0 {
		fmt.Fprintln(os.Stderr, "benchreport: at least one --metrics-file or --metrics-url is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse each snapshot.
	var parsed []snapshot
	for _, s := range snaps {
		fams, err := s.load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchreport: load %q: %v\n", s.src, err)
			os.Exit(1)
		}
		parsed = append(parsed, snapshot{impl: s.impl, families: fams})
	}

	// Build the report.
	var sb strings.Builder
	sb.WriteString("# Gateway Auth Latency Benchmark Report\n\n")

	metricDefs := []struct {
		name      string
		labelKey  string
		title     string
	}{
		{"ecp_gateway_auth_middleware_duration_seconds", "provider", "End-to-end auth middleware latency (`ecp_gateway_auth_middleware_duration_seconds`)"},
		{"ecp_gateway_authz_check_duration_seconds", "impl", "Authorization check latency (`ecp_gateway_authz_check_duration_seconds`)"},
		{"ecp_gateway_rbac_fetch_duration_seconds", "impl", "RBAC data fetch latency (`ecp_gateway_rbac_fetch_duration_seconds`)"},
	}

	for _, md := range metricDefs {
		sb.WriteString("## " + md.title + "\n\n")
		var rows []histStats
		for _, snap := range parsed {
			fam, ok := snap.families[md.name]
			if !ok {
				continue
			}
			for _, m := range fam.GetMetric() {
				labelVal := labelValue(m.GetLabel(), md.labelKey)
				tag := snap.impl
				if labelVal != "" && labelVal != snap.impl {
					tag = snap.impl + "/" + labelVal
				}
				h := parseHistogram(m.GetHistogram(), tag)
				rows = append(rows, h)
			}
		}
		if len(rows) == 0 {
			sb.WriteString("_No data found for this metric._\n\n")
			continue
		}
		sb.WriteString("| impl/label | count | avg (ms) | p50 (ms) | p90 (ms) | p99 (ms) |\n")
		sb.WriteString("|---|---:|---:|---:|---:|---:|\n")
		for _, r := range rows {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.3f | %.3f | %.3f | %.3f |\n",
				r.impl,
				r.count,
				r.avg()*1000,
				r.p50*1000,
				r.p90*1000,
				r.p99*1000,
			))
		}
		sb.WriteString("\n")
	}

	if *testJSON != "" {
		section, err := parseTestJSON(*testJSON)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchreport: parse test-json %q: %v\n", *testJSON, err)
		} else {
			sb.WriteString("## Test Results\n\n")
			sb.WriteString(section)
		}
	}

	if err := writeReport(*out, sb.String()); err != nil {
		fmt.Fprintf(os.Stderr, "benchreport: write report: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "benchreport: wrote %s\n", *out)
}

// snapArg is a (impl, src) pair where src is a file path or URL.
type snapArg struct {
	impl string
	src  string
}

func (s snapArg) load() (map[string]*dto.MetricFamily, error) {
	var r io.Reader
	if strings.HasPrefix(s.src, "http://") || strings.HasPrefix(s.src, "https://") {
		resp, err := http.Get(s.src) //nolint:gosec,noctx
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		r = resp.Body
	} else {
		f, err := os.Open(s.src) //nolint:gosec
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	p := expfmt.NewTextParser(model.LegacyValidation)
	return p.TextToMetricFamilies(r)
}

// parseSnapshotArgs extracts (impl, file/url) pairs from argv.
// The form is: [--impl=<tag>] (--metrics-file=<path> | --metrics-url=<url>)
func parseSnapshotArgs(argv []string) ([]snapArg, error) {
	var result []snapArg
	impl := "default"
	for _, arg := range argv {
		switch {
		case strings.HasPrefix(arg, "--impl="):
			impl = strings.TrimPrefix(arg, "--impl=")
		case strings.HasPrefix(arg, "--metrics-file="):
			result = append(result, snapArg{impl: impl, src: strings.TrimPrefix(arg, "--metrics-file=")})
			impl = "default"
		case strings.HasPrefix(arg, "--metrics-url="):
			result = append(result, snapArg{impl: impl, src: strings.TrimPrefix(arg, "--metrics-url=")})
			impl = "default"
		}
	}
	return result, nil
}

// labelValue returns the value of the first label matching key.
func labelValue(pairs []*dto.LabelPair, key string) string {
	for _, lp := range pairs {
		if lp.GetName() == key {
			return lp.GetValue()
		}
	}
	return ""
}

// parseHistogram computes statistics from a Prometheus histogram.
func parseHistogram(h *dto.Histogram, impl string) histStats {
	if h == nil {
		return histStats{impl: impl}
	}
	count := h.GetSampleCount()
	sum := h.GetSampleSum()
	stats := histStats{
		impl:  impl,
		count: count,
		sum:   sum,
		p50:   interpolate(h, 0.50),
		p90:   interpolate(h, 0.90),
		p99:   interpolate(h, 0.99),
	}
	return stats
}

// interpolate estimates a quantile from cumulative histogram buckets using
// linear interpolation within the bucket containing the target rank.
func interpolate(h *dto.Histogram, q float64) float64 {
	total := float64(h.GetSampleCount())
	if total == 0 {
		return 0
	}
	target := q * total

	buckets := h.GetBucket()
	// Buckets are ordered by upper bound. The last implicit bucket has upper bound +Inf.
	// We need to add the +Inf bucket if it's not present.

	// Sort by upper bound just in case.
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].GetUpperBound() < buckets[j].GetUpperBound()
	})

	var prevCount float64
	var prevUB float64
	for _, b := range buckets {
		cum := float64(b.GetCumulativeCount())
		ub := b.GetUpperBound()
		if math.IsInf(ub, 1) {
			break
		}
		if cum >= target {
			// Target falls in (prevUB, ub].
			width := ub - prevUB
			inBucket := target - prevCount
			bucketCount := cum - prevCount
			if bucketCount == 0 {
				return prevUB
			}
			return prevUB + width*(inBucket/bucketCount)
		}
		prevCount = cum
		prevUB = ub
	}
	// Falls in the last bucket — return its upper bound as an approximation.
	if len(buckets) > 0 {
		return buckets[len(buckets)-1].GetUpperBound()
	}
	return 0
}

// parseTestJSON reads go test -json output and returns a markdown summary section.
func parseTestJSON(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer f.Close()

	type event struct {
		Action  string `json:"Action"`
		Test    string `json:"Test"`
		Package string `json:"Package"`
		Output  string `json:"Output"`
	}

	totals := map[string]int{"pass": 0, "fail": 0, "skip": 0}
	var failed []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.Test == "" {
			continue // package-level event
		}
		switch e.Action {
		case "pass":
			totals["pass"]++
		case "fail":
			totals["fail"]++
			failed = append(failed, fmt.Sprintf("%s/%s", e.Package, e.Test))
		case "skip":
			totals["skip"]++
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **Pass**: %d  **Fail**: %d  **Skip**: %d\n", totals["pass"], totals["fail"], totals["skip"]))
	if len(failed) > 0 {
		sb.WriteString("\n**Failed tests:**\n\n")
		for _, f := range failed {
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}
	sb.WriteString("\n")
	return sb.String(), nil
}

// writeReport creates parent directories as needed and writes content to path.
func writeReport(path, content string) error {
	dir := path[:strings.LastIndex(path, "/")]
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return os.WriteFile(path, []byte(content), 0o640)
}
