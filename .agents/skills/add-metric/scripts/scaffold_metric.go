package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repository root (go.mod) not found in current directory or any parent")
}

func main() {
	var (
		domainFlag  = flag.String("domain", "", "Metric domain: time, quality, code, team")
		metricFlag  = flag.String("metric", "", "Metric name (e.g. security, hotspots, latency)")
		descFlag    = flag.String("desc", "", "Short description of the metric")
		unitFlag    = flag.String("unit", "seconds", "Metric unit: seconds, count, percent, lines")
		checkMetric = flag.String("check", "", "Verify if an existing metric is wired up across all 5 code locations")
	)
	flag.Parse()

	if *checkMetric != "" {
		checkMetricWiring(*checkMetric)
		return
	}

	if *domainFlag == "" || *metricFlag == "" {
		fmt.Fprintf(os.Stderr, "Usage: scaffold_metric --domain <time|quality|code|team> --metric <name> [--desc <desc>] [--unit <unit>]\n")
		fmt.Fprintf(os.Stderr, "       scaffold_metric --check <metric_name>\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	validDomains := map[string]bool{"time": true, "quality": true, "code": true, "team": true}
	domain := strings.ToLower(*domainFlag)
	if !validDomains[domain] {
		fmt.Fprintf(os.Stderr, "Error: Invalid domain %q. Supported domains: time, quality, code, team\n", domain)
		os.Exit(1)
	}

	metric := strings.ToLower(*metricFlag)
	desc := *descFlag
	if desc == "" {
		desc = fmt.Sprintf("Calculates %s metrics for pull requests", metric)
	}
	unit := *unitFlag

	generateScaffolding(domain, metric, desc, unit)
}

func checkMetricWiring(metric string) {
	fmt.Printf("Auditing pipeline wiring for metric %q...\n\n", metric)
	m := strings.ToLower(metric)

	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error locating repo root: %v\n", err)
		os.Exit(1)
	}

	checks := []struct {
		file     string
		desc     string
		patterns []string
	}{
		{
			file:     filepath.Join(root, "pkg/metrics/types.go"),
			desc:     "Type schema definition (ProcessedPR field)",
			patterns: []string{m},
		},
		{
			file:     filepath.Join(root, "pkg/metrics/calculator.go"),
			desc:     "Calculation logic in ProcessPRNode",
			patterns: []string{m},
		},
		{
			file:     filepath.Join(root, "pkg/metrics/aggregation.go"),
			desc:     "Aggregation case in computeMetricStats",
			patterns: []string{fmt.Sprintf(`case "%s":`, m), fmt.Sprintf(`case "%s"`, m)},
		},
		{
			file:     filepath.Join(root, "pkg/cmd/"+getDomainForMetric(m)+".go"),
			desc:     "Cobra subcommand registration",
			patterns: []string{fmt.Sprintf(`RunMetricCommand(cmd, "%s", "%s")`, getDomainForMetric(m), m), fmt.Sprintf(`"%s"`, m)},
		},
	}

	allPassed := true
	for _, c := range checks {
		content, err := os.ReadFile(c.file)
		if err != nil {
			rel, _ := filepath.Rel(root, c.file)
			fmt.Printf("✗ %s (%s): file unreadable: %v\n", c.desc, rel, err)
			allPassed = false
			continue
		}

		found := false
		lowerContent := strings.ToLower(string(content))
		for _, p := range c.patterns {
			if strings.Contains(lowerContent, strings.ToLower(p)) {
				found = true
				break
			}
		}

		rel, _ := filepath.Rel(root, c.file)
		if found {
			fmt.Printf("✓ %s (%s)\n", c.desc, rel)
		} else {
			fmt.Printf("✗ %s (%s): missing metric reference\n", c.desc, rel)
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Printf("Result: Metric %q appears fully wired across all checked locations!\n", metric)
	} else {
		fmt.Printf("Result: Metric %q is missing from one or more pipeline stages.\n", metric)
	}
}

func getDomainForMetric(m string) string {
	switch m {
	case "merge", "review", "draft", "pickup", "idle":
		return "time"
	case "ci", "rework", "conflicts", "reverts":
		return "quality"
	case "size", "commits":
		return "code"
	case "throughput", "reviews", "unreviewed":
		return "team"
	default:
		return "time"
	}
}

func titleCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func generateScaffolding(domain, metric, desc, unit string) {
	pascalMetric := titleCase(metric)

	fmt.Printf("========================================================================\n")
	fmt.Printf("Scaffolding Boilerplate for Metric: %s %s\n", domain, metric)
	fmt.Printf("========================================================================\n\n")

	fmt.Printf("--- 1. pkg/metrics/types.go ---\n")
	fmt.Printf("// Add to ProcessedPR struct:\n")
	if unit == "seconds" {
		fmt.Printf("\t%sDurationSeconds *float64 `json:\"%s_duration_seconds,omitempty\"`\n\n", pascalMetric, metric)
	} else if unit == "count" {
		fmt.Printf("\t%sCount int `json:\"%s_count\"`\n\n", pascalMetric, metric)
	} else if unit == "percent" {
		fmt.Printf("\t%sRate float64 `json:\"%s_rate\"`\n\n", pascalMetric, metric)
	} else {
		fmt.Printf("\t%s float64 `json:\"%s\"`\n\n", pascalMetric, metric)
	}

	fmt.Printf("--- 2. pkg/metrics/calculator.go ---\n")
	fmt.Printf("// In ProcessPRNode(node api.GraphQLPRNode) ProcessedPR:\n")
	fmt.Printf("\t// TODO: Calculate %s metric\n", metric)
	if unit == "seconds" {
		fmt.Printf("\t// Example duration calculation:\n")
		fmt.Printf("\t// duration := node.SomeEnd.Sub(node.SomeStart).Seconds()\n")
		fmt.Printf("\t// pr.%sDurationSeconds = &duration\n\n", pascalMetric)
	} else if unit == "count" {
		fmt.Printf("\t// pr.%sCount = len(node.SomeItems)\n\n", pascalMetric)
	} else {
		fmt.Printf("\t// pr.%s = ...\n\n", pascalMetric)
	}

	fmt.Printf("--- 3. pkg/metrics/aggregation.go ---\n")
	fmt.Printf("// In computeMetricStats, under switch domain -> case %q -> switch metric:\n", domain)
	fmt.Printf("\t\tcase %q:\n", metric)
	fmt.Printf("\t\t\tunit = %q\n", unit)
	fmt.Printf("\t\t\tfor _, pr := range prs {\n")
	if unit == "seconds" {
		fmt.Printf("\t\t\t\tif pr.%sDurationSeconds != nil {\n", pascalMetric)
		fmt.Printf("\t\t\t\t\tvalues = append(values, *pr.%sDurationSeconds)\n", pascalMetric)
		fmt.Printf("\t\t\t\t}\n")
	} else if unit == "count" {
		fmt.Printf("\t\t\t\tvalues = append(values, float64(pr.%sCount))\n", pascalMetric)
	} else {
		fmt.Printf("\t\t\t\tvalues = append(values, pr.%s)\n", pascalMetric)
	}
	fmt.Printf("\t\t\t}\n\n")

	fmt.Printf("--- 4. pkg/cmd/%s.go ---\n", domain)
	fmt.Printf("var %s%sCmd = &cobra.Command{\n", domain, pascalMetric)
	fmt.Printf("\tUse:   %q,\n", metric)
	fmt.Printf("\tShort: %q,\n", desc)
	fmt.Printf("\tLong:  `%s.`,\n", desc)
	fmt.Printf("\tExample: `  # Analyze %s over the past 30 days\n", metric)
	fmt.Printf("  gh pr-pro %s %s --past 30d\n\n", domain, metric)
	fmt.Printf("  # Group %s by author\n", metric)
	fmt.Printf("  gh pr-pro %s %s --past 90d --group-by author`,\n", domain, metric)
	fmt.Printf("\tRunE: func(cmd *cobra.Command, args []string) error {\n")
	fmt.Printf("\t\treturn RunMetricCommand(cmd, %q, %q)\n", domain, metric)
	fmt.Printf("\t},\n}\n\n")
	fmt.Printf("// In init():\n")
	fmt.Printf("\t%sCmd.AddCommand(%s%sCmd)\n\n", domain, domain, pascalMetric)

	fmt.Printf("--- 5. Unit Test Template (pkg/metrics/calculator_test.go) ---\n")
	fmt.Printf("func TestProcessPRNode_%s(t *testing.T) {\n", pascalMetric)
	fmt.Printf("\t// Table-driven test testing %s calculation\n", metric)
	fmt.Printf("\tnode := api.GraphQLPRNode{\n\t\tNumber: 1,\n\t\tTitle:  \"Test %s\",\n\t\tState:  \"MERGED\",\n\t}\n", metric)
	fmt.Printf("\tpr := ProcessPRNode(node)\n")
	fmt.Printf("\t// Add assertions for pr.%s\n", pascalMetric)
	fmt.Printf("}\n\n")

	fmt.Printf("========================================================================\n")
	fmt.Printf("Next Steps:\n")
	fmt.Printf("1. Paste and adjust snippets in the 4 source files above.\n")
	fmt.Printf("2. Run tests: go test -v -race ./pkg/metrics/...\n")
	fmt.Printf("3. Sync README: go run .agents/skills/readme-api-sync/scripts/audit_readme.go\n")
	fmt.Printf("========================================================================\n")
}
