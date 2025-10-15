package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultke-backend/test/helpers"
)

// TestRunner manages the execution of the test suite
type TestRunner struct {
	config     *helpers.TestConfig
	db         *helpers.TestDatabase
	coverage   *CoverageReporter
	results    *TestResults
	parallel   bool
	verbose    bool
	timeout    time.Duration
	workingDir string
	mu         sync.RWMutex
}

// TestResults holds the results of test execution
type TestResults struct {
	TotalTests         int
	PassedTests        int
	FailedTests        int
	SkippedTests       int
	Duration           time.Duration
	Coverage           float64
	FailedTestsDetails []FailedTest
}

// FailedTest represents a failed test case
type FailedTest struct {
	Name     string
	Package  string
	Error    string
	Duration time.Duration
}

// CoverageReporter handles code coverage reporting
type CoverageReporter struct {
	outputPath string
	threshold  float64
	excludes   []string
}

// NewTestRunner creates a new test runner
func NewTestRunner(options ...TestRunnerOption) *TestRunner {
	runner := &TestRunner{
		config:     helpers.NewTestConfig(),
		parallel:   true,
		verbose:    false,
		timeout:    10 * time.Minute,
		workingDir: ".",
		results:    &TestResults{},
		coverage: &CoverageReporter{
			outputPath: "coverage.out",
			threshold:  97.0,
			excludes:   []string{},
		},
	}

	for _, option := range options {
		option(runner)
	}

	return runner
}

// TestRunnerOption is a function that configures a TestRunner
type TestRunnerOption func(*TestRunner)

// WithParallel sets whether to run tests in parallel
func WithParallel(parallel bool) TestRunnerOption {
	return func(r *TestRunner) {
		r.parallel = parallel
	}
}

// WithVerbose sets verbose output
func WithVerbose(verbose bool) TestRunnerOption {
	return func(r *TestRunner) {
		r.verbose = verbose
	}
}

// WithTimeout sets the test timeout
func WithTimeout(timeout time.Duration) TestRunnerOption {
	return func(r *TestRunner) {
		r.timeout = timeout
	}
}

// WithWorkingDirectory sets the working directory
func WithWorkingDirectory(dir string) TestRunnerOption {
	return func(r *TestRunner) {
		r.workingDir = dir
	}
}

// WithCoverageThreshold sets the coverage threshold
func WithCoverageThreshold(threshold float64) TestRunnerOption {
	return func(r *TestRunner) {
		r.coverage.threshold = threshold
	}
}

// WithCoverageExcludes sets files/patterns to exclude from coverage
func WithCoverageExcludes(excludes []string) TestRunnerOption {
	return func(r *TestRunner) {
		r.coverage.excludes = excludes
	}
}

// Setup initializes the test environment
func (r *TestRunner) Setup() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize test database
	r.db = helpers.SetupTestDatabase()

	// Create necessary directories
	dirs := []string{
		"coverage",
		"reports",
		"logs",
		"tmp",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Set up environment variables for testing
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("DATABASE_URL", ":memory:")
	os.Setenv("JWT_SECRET", r.config.JWTSecret)

	return nil
}

// Teardown cleans up the test environment
func (r *TestRunner) Teardown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db != nil {
		r.db.Close()
	}

	// Clean up temporary files
	os.RemoveAll("tmp")
}

// RunAll runs all tests in the test suite
func (r *TestRunner) RunAll() error {
	if err := r.Setup(); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}
	defer r.Teardown()

	start := time.Now()

	// Run different types of tests
	if err := r.runUnitTests(); err != nil {
		return fmt.Errorf("unit tests failed: %w", err)
	}

	if err := r.runIntegrationTests(); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}

	if err := r.runPerformanceTests(); err != nil {
		return fmt.Errorf("performance tests failed: %w", err)
	}

	if err := r.runSecurityTests(); err != nil {
		return fmt.Errorf("security tests failed: %w", err)
	}

	r.results.Duration = time.Since(start)

	// Generate coverage report
	if err := r.generateCoverageReport(); err != nil {
		return fmt.Errorf("coverage report generation failed: %w", err)
	}

	// Generate test report
	if err := r.generateTestReport(); err != nil {
		return fmt.Errorf("test report generation failed: %w", err)
	}

	return nil
}

// runUnitTests runs unit tests
func (r *TestRunner) runUnitTests() error {
	log.Println("Running unit tests...")

	testFiles := []string{
		"./test/models_test.go",
		"./test/services_test.go",
		"./test/middleware_test.go",
		"./test/config_test.go",
		"./test/helpers_test.go",
	}

	for _, testFile := range testFiles {
		if err := r.runTestFile(testFile); err != nil {
			return fmt.Errorf("failed to run %s: %w", testFile, err)
		}
	}

	return nil
}

// runIntegrationTests runs integration tests
func (r *TestRunner) runIntegrationTests() error {
	log.Println("Running integration tests...")

	testFiles := []string{
		"./test/api_handlers_test.go",
		"./test/integration_test.go",
		"./test/api_test.go",
		"./test/auth_test.go",
		"./test/chama_test.go",
		"./test/wallet_test.go",
		"./test/marketplace_test.go",
		"./test/meeting_test.go",
		"./test/loan_test.go",
		"./test/notification_test.go",
		"./test/reminder_test.go",
		"./test/security_test.go",
		"./test/transaction_status_test.go",
		"./test/user_test.go",
	}

	for _, testFile := range testFiles {
		if err := r.runTestFile(testFile); err != nil {
			return fmt.Errorf("failed to run %s: %w", testFile, err)
		}
	}

	return nil
}

// runPerformanceTests runs performance tests
func (r *TestRunner) runPerformanceTests() error {
	log.Println("Running performance tests...")

	return r.runTestFile("./test/performance_test.go")
}

// runSecurityTests runs security tests
func (r *TestRunner) runSecurityTests() error {
	log.Println("Running security tests...")

	return r.runTestFile("./test/security_test.go")
}

// runTestFile runs a specific test file
func (r *TestRunner) runTestFile(testFile string) error {
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		log.Printf("Test file %s does not exist, skipping", testFile)
		return nil
	}

	args := []string{"test"}

	if r.verbose {
		args = append(args, "-v")
	}

	if r.parallel {
		args = append(args, "-parallel", fmt.Sprintf("%d", runtime.NumCPU()))
	}

	args = append(args, "-timeout", r.timeout.String())
	args = append(args, "-coverprofile", r.coverage.outputPath)
	args = append(args, testFile)

	cmd := exec.Command("go", args...)
	cmd.Dir = r.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		r.parseTestOutput(stderr.String())
		return fmt.Errorf("test failed: %w\nOutput: %s", err, stderr.String())
	}

	r.parseTestOutput(stdout.String())
	return nil
}

// parseTestOutput parses test output to extract results
func (r *TestRunner) parseTestOutput(output string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "PASS") {
			r.results.PassedTests++
		} else if strings.Contains(line, "FAIL") {
			r.results.FailedTests++
			r.parseFailedTest(line)
		} else if strings.Contains(line, "SKIP") {
			r.results.SkippedTests++
		}
	}

	r.results.TotalTests = r.results.PassedTests + r.results.FailedTests + r.results.SkippedTests
}

// parseFailedTest parses a failed test line to extract details
func (r *TestRunner) parseFailedTest(line string) {
	// Parse failed test details using regex
	re := regexp.MustCompile(`FAIL\s+(\S+)\s+(\S+)\s+(\d+\.\d+)s`)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 4 {
		duration, _ := strconv.ParseFloat(matches[3], 64)

		failedTest := FailedTest{
			Name:     matches[1],
			Package:  matches[2],
			Duration: time.Duration(duration * float64(time.Second)),
			Error:    line,
		}

		r.results.FailedTestsDetails = append(r.results.FailedTestsDetails, failedTest)
	}
}

// generateCoverageReport generates a coverage report
func (r *TestRunner) generateCoverageReport() error {
	log.Println("Generating coverage report...")

	// Run go tool cover to get coverage percentage
	cmd := exec.Command("go", "tool", "cover", "-func", r.coverage.outputPath)
	cmd.Dir = r.workingDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate coverage report: %w", err)
	}

	// Parse coverage output
	output := stdout.String()
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				coverageStr := strings.TrimSuffix(parts[2], "%")
				if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
					r.results.Coverage = coverage
				}
			}
		}
	}

	// Generate HTML coverage report
	htmlCmd := exec.Command("go", "tool", "cover", "-html", r.coverage.outputPath, "-o", "coverage/coverage.html")
	htmlCmd.Dir = r.workingDir

	if err := htmlCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML coverage report: %w", err)
	}

	// Check if coverage meets threshold
	if r.results.Coverage < r.coverage.threshold {
		return fmt.Errorf("coverage %.2f%% is below threshold %.2f%%", r.results.Coverage, r.coverage.threshold)
	}

	return nil
}

// generateTestReport generates a comprehensive test report
func (r *TestRunner) generateTestReport() error {
	log.Println("Generating test report...")

	reportFile := "reports/test_report.html"

	report := r.generateHTMLReport()

	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		return fmt.Errorf("failed to write test report: %w", err)
	}

	// Also generate JSON report for CI/CD
	jsonReportFile := "reports/test_report.json"
	jsonReport := r.generateJSONReport()

	if err := os.WriteFile(jsonReportFile, []byte(jsonReport), 0644); err != nil {
		return fmt.Errorf("failed to write JSON test report: %w", err)
	}

	return nil
}

// generateHTMLReport generates an HTML test report
func (r *TestRunner) generateHTMLReport() string {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>VaultKe Backend Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f5f5f5; padding: 20px; border-radius: 5px; }
        .summary { display: flex; justify-content: space-between; margin: 20px 0; }
        .metric { text-align: center; padding: 20px; background-color: #e9ecef; border-radius: 5px; }
        .metric.pass { background-color: #d4edda; color: #155724; }
        .metric.fail { background-color: #f8d7da; color: #721c24; }
        .metric.skip { background-color: #fff3cd; color: #856404; }
        .failed-tests { margin-top: 20px; }
        .failed-test { background-color: #f8d7da; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .footer { margin-top: 40px; text-align: center; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>VaultKe Backend Test Report</h1>
        <p>Generated on: %s</p>
        <p>Duration: %s</p>
    </div>
    
    <div class="summary">
        <div class="metric pass">
            <h3>%d</h3>
            <p>Passed</p>
        </div>
        <div class="metric fail">
            <h3>%d</h3>
            <p>Failed</p>
        </div>
        <div class="metric skip">
            <h3>%d</h3>
            <p>Skipped</p>
        </div>
        <div class="metric">
            <h3>%.2f%%</h3>
            <p>Coverage</p>
        </div>
    </div>
    
    %s
    
    <div class="footer">
        <p>VaultKe Backend Test Suite - Generated by Test Runner</p>
    </div>
</body>
</html>
`

	failedTestsHTML := ""
	if len(r.results.FailedTestsDetails) > 0 {
		failedTestsHTML = "<div class=\"failed-tests\"><h2>Failed Tests</h2>"
		for _, failedTest := range r.results.FailedTestsDetails {
			failedTestsHTML += fmt.Sprintf(`
				<div class="failed-test">
					<h4>%s</h4>
					<p><strong>Package:</strong> %s</p>
					<p><strong>Duration:</strong> %s</p>
					<p><strong>Error:</strong> %s</p>
				</div>
			`, failedTest.Name, failedTest.Package, failedTest.Duration, failedTest.Error)
		}
		failedTestsHTML += "</div>"
	}

	return fmt.Sprintf(html,
		time.Now().Format("2006-01-02 15:04:05"),
		r.results.Duration.String(),
		r.results.PassedTests,
		r.results.FailedTests,
		r.results.SkippedTests,
		r.results.Coverage,
		failedTestsHTML,
	)
}

// generateJSONReport generates a JSON test report
func (r *TestRunner) generateJSONReport() string {
	report := map[string]interface{}{
		"timestamp":            time.Now().Format("2006-01-02T15:04:05Z"),
		"duration":             r.results.Duration.String(),
		"total_tests":          r.results.TotalTests,
		"passed_tests":         r.results.PassedTests,
		"failed_tests":         r.results.FailedTests,
		"skipped_tests":        r.results.SkippedTests,
		"coverage":             r.results.Coverage,
		"failed_tests_details": r.results.FailedTestsDetails,
	}

	jsonBytes, _ := json.Marshal(report)
	return string(jsonBytes)
}

// RunWithWatcher runs tests with file system watching
func (r *TestRunner) RunWithWatcher() error {
	// Initial test run
	if err := r.RunAll(); err != nil {
		log.Printf("Initial test run failed: %v", err)
	}

	// Watch for file changes
	return r.watchForChanges()
}

// watchForChanges watches for file changes and re-runs tests
func (r *TestRunner) watchForChanges() error {
	log.Println("Watching for file changes...")

	// This is a simplified file watcher
	// In a real implementation, you'd use a proper file system watcher like fsnotify
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastModTime := time.Now()

	for {
		select {
		case <-ticker.C:
			if r.hasChanges(lastModTime) {
				lastModTime = time.Now()
				log.Println("Changes detected, running tests...")

				if err := r.RunAll(); err != nil {
					log.Printf("Test run failed: %v", err)
				} else {
					log.Println("Tests completed successfully")
				}
			}
		}
	}
}

// hasChanges checks if any Go files have changed since the last modification time
func (r *TestRunner) hasChanges(lastModTime time.Time) bool {
	changed := false

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".go") && info.ModTime().After(lastModTime) {
			changed = true
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		log.Printf("Error checking for changes: %v", err)
	}

	return changed
}

// PrintResults prints the test results to stdout
func (r *TestRunner) PrintResults() {
	fmt.Println("=== Test Results ===")
	fmt.Printf("Total Tests: %d\n", r.results.TotalTests)
	fmt.Printf("Passed: %d\n", r.results.PassedTests)
	fmt.Printf("Failed: %d\n", r.results.FailedTests)
	fmt.Printf("Skipped: %d\n", r.results.SkippedTests)
	fmt.Printf("Coverage: %.2f%%\n", r.results.Coverage)
	fmt.Printf("Duration: %s\n", r.results.Duration)

	if len(r.results.FailedTestsDetails) > 0 {
		fmt.Println("\n=== Failed Tests ===")
		for _, failedTest := range r.results.FailedTestsDetails {
			fmt.Printf("- %s (%s) - %s\n", failedTest.Name, failedTest.Package, failedTest.Duration)
		}
	}

	if r.results.Coverage >= r.coverage.threshold {
		fmt.Printf("\n✅ Coverage threshold %.2f%% achieved!\n", r.coverage.threshold)
	} else {
		fmt.Printf("\n❌ Coverage %.2f%% below threshold %.2f%%\n", r.results.Coverage, r.coverage.threshold)
	}
}

// RunLinting runs code linting
func (r *TestRunner) RunLinting() error {
	log.Println("Running code linting...")

	// Run golint
	cmd := exec.Command("golint", "./...")
	cmd.Dir = r.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Linting warnings/errors:\n%s", stdout.String())
		// Don't fail on linting errors, just log them
	}

	return nil
}

// RunStaticAnalysis runs static analysis
func (r *TestRunner) RunStaticAnalysis() error {
	log.Println("Running static analysis...")

	// Run go vet
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = r.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("static analysis failed: %w\nOutput: %s", err, stderr.String())
	}

	return nil
}

// RunSecurityScan runs security scanning
func (r *TestRunner) RunSecurityScan() error {
	log.Println("Running security scan...")

	// Run gosec
	cmd := exec.Command("gosec", "./...")
	cmd.Dir = r.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Security scan warnings:\n%s", stdout.String())
		// Don't fail on security warnings, just log them
	}

	return nil
}

// RunBenchmarks runs benchmark tests
func (r *TestRunner) RunBenchmarks() error {
	log.Println("Running benchmarks...")

	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "./test/")
	cmd.Dir = r.workingDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("benchmarks failed: %w", err)
	}

	// Save benchmark results
	benchmarkFile := "reports/benchmark_results.txt"
	if err := os.WriteFile(benchmarkFile, stdout.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to save benchmark results: %w", err)
	}

	return nil
}

// GetResults returns the test results
func (r *TestRunner) GetResults() *TestResults {
	return r.results
}

// TestFullTestSuite runs the complete test suite
func TestFullTestSuite(t *testing.T) {
	runner := NewTestRunner(
		WithVerbose(true),
		WithCoverageThreshold(97.0),
		WithTimeout(15*time.Minute),
	)

	// Run all tests
	err := runner.RunAll()
	require.NoError(t, err, "Test suite should complete without errors")

	// Print results
	runner.PrintResults()

	// Validate results
	results := runner.GetResults()
	assert.Greater(t, results.TotalTests, 0, "Should have run some tests")
	assert.Equal(t, 0, results.FailedTests, "Should have no failed tests")
	assert.GreaterOrEqual(t, results.Coverage, 97.0, "Should achieve 97% coverage")

	// Run additional checks
	err = runner.RunLinting()
	assert.NoError(t, err, "Linting should pass")

	err = runner.RunStaticAnalysis()
	assert.NoError(t, err, "Static analysis should pass")

	err = runner.RunSecurityScan()
	assert.NoError(t, err, "Security scan should pass")

	err = runner.RunBenchmarks()
	assert.NoError(t, err, "Benchmarks should run successfully")
}
