package smoke

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// Runner executes scenarios by invoking the cora binary as a subprocess.
type Runner struct {
	coraBin       string // path to cora binary
	configPath    string // expanded config file path (may be empty)
	verbose       bool   // pass --verbose to every cora invocation
	maxConcurrent int    // max parallel service groups (0 = sequential)
}

// NewRunner creates a Runner. configPath may be "" to skip CORA_CONFIG injection.
// maxConcurrent controls parallel execution: 0 = sequential, >0 = run up to N
// service groups concurrently.
func NewRunner(coraBin, configPath string, verbose bool, maxConcurrent int) *Runner {
	return &Runner{
		coraBin: coraBin, configPath: configPath, verbose: verbose,
		maxConcurrent: maxConcurrent,
	}
}

// Run executes a single Scenario and returns its result.
func (r *Runner) Run(s Scenario) ScenarioResult {
	result := ScenarioResult{Scenario: s}

	if s.Skip {
		result.Status = StatusSkip
		return result
	}

	// Expand env vars in args (allows ${SMOKE_GITCODE_OWNER} in scenario files).
	expandedArgs := make([]string, len(s.Args))
	for i, arg := range s.Args {
		expandedArgs[i] = os.ExpandEnv(arg)
	}

	// Build command: <service> <args...> --format <format> [--verbose]
	args := append([]string{s.Service}, expandedArgs...)
	args = append(args, "--format", s.Format)
	if r.verbose {
		args = append(args, "--verbose")
	}

	if r.verbose {
		fmt.Printf("  [cmd] %s %s\n", r.coraBin, strings.Join(args, " "))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.TimeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.coraBin, args...)

	// Inject expanded smoke config via CORA_CONFIG env var.
	cmd.Env = os.Environ()
	if r.configPath != "" {
		cmd.Env = append(cmd.Env, "CORA_CONFIG="+r.configPath)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	result.DurationMs = time.Since(start).Milliseconds()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = StatusTimeout
		result.Err = fmt.Sprintf("timed out after %dms", s.TimeoutMs)
		return result
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Status = StatusError
			result.Err = fmt.Sprintf("subprocess error: %v", err)
			return result
		}
	}

	// Evaluate assertions.
	allPass := true
	for _, a := range s.Assertions {
		ar := EvaluateAssertion(a, result.Stdout, result.Stderr, result.ExitCode, result.DurationMs)
		result.AssertionResults = append(result.AssertionResults, ar)
		if !ar.Passed {
			allPass = false
		}
	}

	if allPass {
		result.Status = StatusPass
	} else {
		result.Status = StatusFail
	}
	return result
}

// RunAll executes all scenarios and returns a RunReport.
// When MaxConcurrent > 0, scenarios are grouped by service and service groups
// run in parallel (up to MaxConcurrent at a time). Scenarios within a group
// still run sequentially to avoid rate-limiting the same API.
func (r *Runner) RunAll(scenarios []Scenario, configPath string) *RunReport {
	if r.maxConcurrent <= 0 {
		return r.runSequential(scenarios, configPath)
	}
	return r.runParallel(scenarios, configPath)
}

// runSequential executes all scenarios one-by-one.
func (r *Runner) runSequential(scenarios []Scenario, configPath string) *RunReport {
	report := &RunReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		ConfigPath:  configPath,
	}
	tStart := time.Now()
	for _, s := range scenarios {
		report.Results = append(report.Results, r.Run(s))
	}
	report.TotalDurationMs = time.Since(tStart).Milliseconds()
	return report
}

// runParallel groups scenarios by service, then runs service groups in
// parallel (up to MaxConcurrent). Scenarios within each group run sequentially
// to avoid overwhelming a single API with concurrent requests.
func (r *Runner) runParallel(scenarios []Scenario, configPath string) *RunReport {
	tStart := time.Now()
	// Group by service, preserving order within each group.
	groups := make(map[string][]Scenario)
	var serviceOrder []string
	for _, s := range scenarios {
		if _, ok := groups[s.Service]; !ok {
			serviceOrder = append(serviceOrder, s.Service)
		}
		groups[s.Service] = append(groups[s.Service], s)
	}

	// Assign each scenario an index for deterministic output ordering.
	origIdx := make(map[string]int)
	for i, s := range scenarios {
		origIdx[s.Name] = i
	}

	type indexedResult struct {
		idx    int
		result ScenarioResult
	}

	resultCh := make(chan indexedResult, len(scenarios))
	sem := make(chan struct{}, r.maxConcurrent)
	var wg sync.WaitGroup

	for _, svc := range serviceOrder {
		wg.Add(1)
		go func(svcName string, svcScenarios []Scenario) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			for _, s := range svcScenarios {
				res := r.Run(s)
				resultCh <- indexedResult{idx: origIdx[s.Name], result: res}
			}
		}(svc, groups[svc])
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect and sort by original index.
	var indexed []indexedResult
	for ir := range resultCh {
		indexed = append(indexed, ir)
	}
	sort.Slice(indexed, func(i, j int) bool { return indexed[i].idx < indexed[j].idx })

	report := &RunReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		ConfigPath:  configPath,
	}
	for _, ir := range indexed {
		report.Results = append(report.Results, ir.result)
	}
	report.TotalDurationMs = time.Since(tStart).Milliseconds()
	return report
}
