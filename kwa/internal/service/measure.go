package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mthesis/kwa/internal/constant"
	"mthesis/kwa/internal/entity"
)

// MeasureRunRequest captures the inputs required to execute one measure.sh run.
type MeasureRunRequest struct {
	Languages  []string
	Benchmarks []string
	Iterations int
}

// MeasureRunner defines the service contract used by app-layer measure orchestration.
type MeasureRunner interface {
	Run(context.Context, MeasureRunRequest) (entity.TimeRangeFilter, error)
}

type runMeasureScriptFunc func(ctx context.Context, scriptPath string, args []string, logPath string) error
type resolveMeasureScriptPathFunc func(cwd string, getenv func(string) string) (string, error)

// MeasureDependencies defines overridable dependencies used by MeasureService for testability.
type MeasureDependencies struct {
	Getwd             func() (string, error)
	Getenv            func(string) string
	Now               func() time.Time
	ResolveScriptPath resolveMeasureScriptPathFunc
	RunScript         runMeasureScriptFunc
}

// MeasureService owns measure.sh execution details and returns the captured
// inclusive time range used by downstream CSV export.
type MeasureService struct {
	deps MeasureDependencies
}

// NewMeasureService returns the production measure service with default dependencies.
func NewMeasureService() *MeasureService {
	return NewMeasureServiceWithDeps(MeasureDependencies{})
}

// NewMeasureServiceWithDeps builds a measure service with dependency overrides
// and falls back to production defaults for missing callbacks.
func NewMeasureServiceWithDeps(deps MeasureDependencies) *MeasureService {
	if deps.Getwd == nil {
		deps.Getwd = os.Getwd
	}
	if deps.Getenv == nil {
		deps.Getenv = os.Getenv
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.ResolveScriptPath == nil {
		deps.ResolveScriptPath = resolveMeasureScriptPath
	}
	if deps.RunScript == nil {
		deps.RunScript = runMeasureScript
	}

	return &MeasureService{deps: deps}
}

// Run resolves script/log paths, executes measure.sh, captures start/end times
// in UTC, and returns the inclusive interval for follow-up export filtering.
func (s *MeasureService) Run(ctx context.Context, req MeasureRunRequest) (entity.TimeRangeFilter, error) {
	if err := validateMeasureRunRequest(req); err != nil {
		return entity.TimeRangeFilter{}, err
	}

	cwd, err := s.deps.Getwd()
	if err != nil {
		return entity.TimeRangeFilter{}, fmt.Errorf("resolve current working directory: %w", err)
	}

	scriptPath, err := s.deps.ResolveScriptPath(cwd, s.deps.Getenv)
	if err != nil {
		return entity.TimeRangeFilter{}, err
	}

	repoRoot := resolveMeasureScriptWorkDir(scriptPath)
	logPath := filepath.Clean(filepath.Join(repoRoot, constant.DefaultMeasureLogPath))
	if err := clearMeasureLogFile(logPath); err != nil {
		return entity.TimeRangeFilter{}, fmt.Errorf("reset measure log file: %w", err)
	}

	startAt := s.deps.Now().UTC()
	measureArgs := buildMeasureScriptArgs(req)
	if err := s.deps.RunScript(ctx, scriptPath, measureArgs, logPath); err != nil {
		return entity.TimeRangeFilter{}, fmt.Errorf("run measure script: %w", err)
	}

	endAt := s.deps.Now().UTC()
	if endAt.Before(startAt) {
		endAt = startAt
	}

	from := startAt
	to := endAt
	filter := entity.TimeRangeFilter{From: &from, To: &to}
	if err := filter.Validate(); err != nil {
		return entity.TimeRangeFilter{}, fmt.Errorf("invalid measure interval: %w", err)
	}

	return filter.Clone(), nil
}

// validateMeasureRunRequest enforces minimum service-level invariants for
// running measure.sh and prevents malformed command construction.
func validateMeasureRunRequest(req MeasureRunRequest) error {
	if req.Iterations <= 0 {
		return fmt.Errorf("iterations must be greater than zero")
	}
	if len(req.Languages) == 0 {
		return fmt.Errorf("at least one language must be selected")
	}
	if len(req.Benchmarks) == 0 {
		return fmt.Errorf("at least one benchmark must be selected")
	}

	for _, language := range req.Languages {
		if strings.TrimSpace(language) == "" {
			return fmt.Errorf("languages must not contain empty values")
		}
	}
	for _, benchmark := range req.Benchmarks {
		if strings.TrimSpace(benchmark) == "" {
			return fmt.Errorf("benchmarks must not contain empty values")
		}
	}

	return nil
}

// buildMeasureScriptArgs maps one run request into scripts/measure.sh key=value
// arguments while preserving caller-provided ordering.
func buildMeasureScriptArgs(req MeasureRunRequest) []string {
	return []string{
		"profile=measure",
		"lang=" + strings.Join(req.Languages, ","),
		"bench=" + strings.Join(req.Benchmarks, ","),
		"iterations=" + strconv.Itoa(req.Iterations),
	}
}

// resolveMeasureScriptPath resolves scripts/measure.sh using deterministic
// precedence: KWA_MEASURE_SCRIPT, KWA_REPO_ROOT/scripts, then upward search
// from cwd to filesystem root.
func resolveMeasureScriptPath(cwd string, getenv func(string) string) (string, error) {
	trimmed := strings.TrimSpace(cwd)
	if trimmed == "" {
		return "", fmt.Errorf("working directory must not be empty")
	}
	if getenv == nil {
		getenv = os.Getenv
	}

	candidates := make([]string, 0, 16)
	addCandidate := func(candidate string) {
		cleaned := filepath.Clean(candidate)
		for _, existing := range candidates {
			if existing == cleaned {
				return
			}
		}
		candidates = append(candidates, cleaned)
	}

	if overrideScript := strings.TrimSpace(getenv("KWA_MEASURE_SCRIPT")); overrideScript != "" {
		candidate := resolvePathFromBase(trimmed, overrideScript)
		addCandidate(candidate)
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	if overrideRepoRoot := strings.TrimSpace(getenv("KWA_REPO_ROOT")); overrideRepoRoot != "" {
		repoRoot := resolvePathFromBase(trimmed, overrideRepoRoot)
		candidate := filepath.Clean(filepath.Join(repoRoot, "scripts", "measure.sh"))
		addCandidate(candidate)
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	for dir := filepath.Clean(trimmed); ; {
		candidate := filepath.Clean(filepath.Join(dir, "scripts", "measure.sh"))
		addCandidate(candidate)
		if fileExists(candidate) {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	quotedCandidates := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		quotedCandidates = append(quotedCandidates, fmt.Sprintf("%q", candidate))
	}

	return "", fmt.Errorf("measure script not found; checked %s", strings.Join(quotedCandidates, ", "))
}

// resolvePathFromBase normalizes raw paths and resolves relative values against baseDir.
func resolvePathFromBase(baseDir, rawPath string) string {
	cleaned := filepath.Clean(strings.TrimSpace(rawPath))
	if filepath.IsAbs(cleaned) {
		return cleaned
	}

	return filepath.Clean(filepath.Join(baseDir, cleaned))
}

// fileExists reports whether path points to an existing regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// clearMeasureLogFile removes the previous measure log file so each run starts
// with a fresh output log.
func clearMeasureLogFile(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("measure log path must not be empty")
	}

	if err := os.Remove(trimmed); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %q: %w", trimmed, err)
	}

	return nil
}

// runMeasureScript executes scripts/measure.sh and appends script stdout/stderr
// to the provided log file path.
func runMeasureScript(ctx context.Context, scriptPath string, args []string, logPath string) error {
	trimmedLogPath := strings.TrimSpace(logPath)
	if trimmedLogPath == "" {
		return fmt.Errorf("measure log path must not be empty")
	}

	if err := os.MkdirAll(filepath.Dir(trimmedLogPath), 0o755); err != nil {
		return fmt.Errorf("create measure log directory for %q: %w", trimmedLogPath, err)
	}

	logFile, err := os.OpenFile(trimmedLogPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open measure log file %q: %w", trimmedLogPath, err)
	}
	defer logFile.Close()

	startOffset, err := logFile.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("seek measure log file %q: %w", trimmedLogPath, err)
	}

	cmd := exec.CommandContext(ctx, scriptPath, args...)
	cmd.Dir = resolveMeasureScriptWorkDir(scriptPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	runErr := cmd.Run()

	logSegmentAnalysis, analysisErr := analyzeMeasureLogSegment(logFile, startOffset)
	if analysisErr != nil {
		if runErr != nil {
			return fmt.Errorf(
				"execute %s (see %s): %w (also failed to inspect log output: %v)",
				scriptPath,
				trimmedLogPath,
				runErr,
				analysisErr,
			)
		}
		return fmt.Errorf("inspect measure log output in %s: %w", trimmedLogPath, analysisErr)
	}

	if runErr != nil {
		summary := selectMeasureFailureSummary(logSegmentAnalysis)
		if summary == "" {
			return fmt.Errorf("execute %s (see %s): %w", scriptPath, trimmedLogPath, runErr)
		}
		return fmt.Errorf("execute %s (see %s): %w: %s", scriptPath, trimmedLogPath, runErr, summary)
	}

	if logSegmentAnalysis.FatalLine != "" {
		return fmt.Errorf("detected fatal output in %s: %s", trimmedLogPath, logSegmentAnalysis.FatalLine)
	}

	return nil
}

// resolveMeasureScriptWorkDir returns the repository root inferred from
// scripts/measure.sh path, matching `make measure` execution context.
func resolveMeasureScriptWorkDir(scriptPath string) string {
	scriptDir := filepath.Dir(scriptPath)
	return filepath.Clean(filepath.Join(scriptDir, ".."))
}

type measureLogSegmentAnalysis struct {
	FatalLine        string
	ErrorLine        string
	LastNonEmptyLine string
}

// analyzeMeasureLogSegment scans only newly appended log lines for one run and
// extracts fatal, generic-error, and last-non-empty summaries.
func analyzeMeasureLogSegment(logFile *os.File, startOffset int64) (measureLogSegmentAnalysis, error) {
	analysis := measureLogSegmentAnalysis{}

	endOffset, err := logFile.Seek(0, io.SeekEnd)
	if err != nil {
		return analysis, fmt.Errorf("seek to end of measure log: %w", err)
	}
	if endOffset <= startOffset {
		return analysis, nil
	}

	sectionReader := io.NewSectionReader(logFile, startOffset, endOffset-startOffset)
	scanner := bufio.NewScanner(sectionReader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		analysis.LastNonEmptyLine = line
		if analysis.FatalLine == "" && isFatalMeasureOutputLine(line) {
			analysis.FatalLine = line
		}

		lowerLine := strings.ToLower(line)
		if analysis.ErrorLine == "" && (strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "exception")) {
			analysis.ErrorLine = line
		}
	}
	if err := scanner.Err(); err != nil {
		return analysis, fmt.Errorf("scan measure log output: %w", err)
	}

	return analysis, nil
}

// selectMeasureFailureSummary returns the highest-priority diagnostic line for
// non-zero script exits: fatal marker, then error/exception, then last line.
func selectMeasureFailureSummary(analysis measureLogSegmentAnalysis) string {
	if analysis.FatalLine != "" {
		return analysis.FatalLine
	}
	if analysis.ErrorLine != "" {
		return analysis.ErrorLine
	}

	return analysis.LastNonEmptyLine
}

// isFatalMeasureOutputLine identifies log lines that indicate a fatal GMT
// measurement failure and should fail the TUI workflow.
func isFatalMeasureOutputLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	fatalMarkers := []string{
		"Final_exception (",
		"Final_exception:",
		"ConfigurationCheckError",
		"Error: Base exception occured in runner.py",
	}
	for _, marker := range fatalMarkers {
		if strings.Contains(trimmed, marker) {
			return true
		}
	}

	return false
}
