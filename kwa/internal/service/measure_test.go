package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMeasureServiceRunSuccess(t *testing.T) {
	t.Parallel()

	var (
		gotScriptPath string
		gotScriptArgs []string
		gotLogPath    string
	)
	nowValues := []time.Time{
		time.Date(2026, time.April, 4, 10, 0, 0, 0, time.Local),
		time.Date(2026, time.April, 4, 10, 5, 0, 0, time.Local),
	}
	nowIndex := 0

	svc := NewMeasureServiceWithDeps(MeasureDependencies{
		Getwd:  func() (string, error) { return "/repo/kwa", nil },
		Getenv: func(string) string { return "" },
		Now: func() time.Time {
			value := nowValues[nowIndex]
			nowIndex++
			return value
		},
		ResolveScriptPath: func(cwd string, _ func(string) string) (string, error) {
			if cwd != "/repo/kwa" {
				t.Fatalf("cwd = %q, want %q", cwd, "/repo/kwa")
			}
			return "/repo/scripts/measure.sh", nil
		},
		RunScript: func(_ context.Context, scriptPath string, args []string, logPath string) error {
			gotScriptPath = scriptPath
			gotScriptArgs = append([]string(nil), args...)
			gotLogPath = logPath
			return nil
		},
	})

	filter, err := svc.Run(context.Background(), MeasureRunRequest{
		Languages:  []string{"go", "c"},
		Benchmarks: []string{"binary-trees", "mandelbrot"},
		Iterations: 3,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if gotScriptPath != "/repo/scripts/measure.sh" {
		t.Fatalf("script path = %q, want %q", gotScriptPath, "/repo/scripts/measure.sh")
	}
	wantArgs := []string{
		"profile=measure",
		"lang=go,c",
		"bench=binary-trees,mandelbrot",
		"iterations=3",
	}
	if !reflect.DeepEqual(gotScriptArgs, wantArgs) {
		t.Fatalf("script args = %#v, want %#v", gotScriptArgs, wantArgs)
	}
	if gotLogPath != "/repo/logs/measure.txt" {
		t.Fatalf("log path = %q, want %q", gotLogPath, "/repo/logs/measure.txt")
	}

	if filter.From == nil || filter.To == nil {
		t.Fatalf("expected non-nil interval")
	}
	if !filter.From.Equal(nowValues[0].UTC()) || !filter.To.Equal(nowValues[1].UTC()) {
		t.Fatalf("unexpected interval: from=%v to=%v", filter.From, filter.To)
	}
}

func TestMeasureServiceRunReturnsScriptErrors(t *testing.T) {
	t.Parallel()

	svc := NewMeasureServiceWithDeps(MeasureDependencies{
		Getwd:             func() (string, error) { return "/repo/kwa", nil },
		Getenv:            func(string) string { return "" },
		Now:               func() time.Time { return time.Date(2026, time.April, 4, 10, 0, 0, 0, time.Local) },
		ResolveScriptPath: func(string, func(string) string) (string, error) { return "/repo/scripts/measure.sh", nil },
		RunScript:         func(context.Context, string, []string, string) error { return errors.New("boom") },
	})

	_, err := svc.Run(context.Background(), MeasureRunRequest{
		Languages:  []string{"go"},
		Benchmarks: []string{"binary-trees"},
		Iterations: 1,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "run measure script") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMeasureServiceRunClearsPreviousMeasureLog(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	repoRoot := filepath.Join(rootDir, "repo")
	logPath := filepath.Join(repoRoot, "logs", "measure.txt")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("stale log\n"), 0o644); err != nil {
		t.Fatalf("seed stale log: %v", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "measure.sh")
	svc := NewMeasureServiceWithDeps(MeasureDependencies{
		Getwd:  func() (string, error) { return filepath.Join(rootDir, "repo", "kwa"), nil },
		Getenv: func(string) string { return "" },
		Now:    func() time.Time { return time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC) },
		ResolveScriptPath: func(string, func(string) string) (string, error) {
			return scriptPath, nil
		},
		RunScript: func(_ context.Context, _ string, _ []string, resolvedLogPath string) error {
			if resolvedLogPath != logPath {
				t.Fatalf("log path = %q, want %q", resolvedLogPath, logPath)
			}
			_, err := os.Stat(logPath)
			if err == nil || !os.IsNotExist(err) {
				t.Fatalf("expected log file to be removed before run, stat err=%v", err)
			}
			return nil
		},
	})

	_, err := svc.Run(context.Background(), MeasureRunRequest{
		Languages:  []string{"go"},
		Benchmarks: []string{"binary-trees"},
		Iterations: 1,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestResolveMeasureScriptPathPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("KWA_MEASURE_SCRIPT takes precedence", func(t *testing.T) {
		rootDir := t.TempDir()
		currentDir := filepath.Join(rootDir, "repo", "kwa", "nested")
		if err := osMkdirAll(currentDir); err != nil {
			t.Fatalf("mkdir current dir: %v", err)
		}

		overrideScript := filepath.Join(rootDir, "custom", "measure.sh")
		repoRootScript := filepath.Join(rootDir, "env-repo", "scripts", "measure.sh")
		upwardSearchScript := filepath.Join(rootDir, "repo", "scripts", "measure.sh")
		for _, path := range []string{overrideScript, repoRootScript, upwardSearchScript} {
			if err := writeExecutableScript(path, "#!/usr/bin/env bash\nexit 0\n"); err != nil {
				t.Fatalf("write script %q: %v", path, err)
			}
		}

		getenv := getenvMap(map[string]string{
			"KWA_MEASURE_SCRIPT": overrideScript,
			"KWA_REPO_ROOT":      filepath.Join(rootDir, "env-repo"),
		})

		resolved, err := resolveMeasureScriptPath(currentDir, getenv)
		if err != nil {
			t.Fatalf("resolve script path: %v", err)
		}
		if resolved != filepath.Clean(overrideScript) {
			t.Fatalf("resolved = %q, want %q", resolved, filepath.Clean(overrideScript))
		}
	})

	t.Run("KWA_REPO_ROOT is second priority", func(t *testing.T) {
		rootDir := t.TempDir()
		currentDir := filepath.Join(rootDir, "repo", "kwa")
		if err := osMkdirAll(currentDir); err != nil {
			t.Fatalf("mkdir current dir: %v", err)
		}

		repoRootScript := filepath.Join(rootDir, "env-repo", "scripts", "measure.sh")
		upwardSearchScript := filepath.Join(rootDir, "repo", "scripts", "measure.sh")
		for _, path := range []string{repoRootScript, upwardSearchScript} {
			if err := writeExecutableScript(path, "#!/usr/bin/env bash\nexit 0\n"); err != nil {
				t.Fatalf("write script %q: %v", path, err)
			}
		}

		getenv := getenvMap(map[string]string{
			"KWA_REPO_ROOT": filepath.Join(rootDir, "env-repo"),
		})

		resolved, err := resolveMeasureScriptPath(currentDir, getenv)
		if err != nil {
			t.Fatalf("resolve script path: %v", err)
		}
		if resolved != filepath.Clean(repoRootScript) {
			t.Fatalf("resolved = %q, want %q", resolved, filepath.Clean(repoRootScript))
		}
	})

	t.Run("falls back to upward search from cwd", func(t *testing.T) {
		rootDir := t.TempDir()
		repoRoot := filepath.Join(rootDir, "project")
		currentDir := filepath.Join(repoRoot, "kwa", "nested", "deeper")
		if err := osMkdirAll(currentDir); err != nil {
			t.Fatalf("mkdir current dir: %v", err)
		}

		upwardSearchScript := filepath.Join(repoRoot, "scripts", "measure.sh")
		if err := writeExecutableScript(upwardSearchScript, "#!/usr/bin/env bash\nexit 0\n"); err != nil {
			t.Fatalf("write upward script: %v", err)
		}

		resolved, err := resolveMeasureScriptPath(currentDir, getenvMap(nil))
		if err != nil {
			t.Fatalf("resolve script path: %v", err)
		}
		if resolved != filepath.Clean(upwardSearchScript) {
			t.Fatalf("resolved = %q, want %q", resolved, filepath.Clean(upwardSearchScript))
		}
	})
}

func TestResolveMeasureScriptPathReturnsClearError(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	currentDir := filepath.Join(rootDir, "repo", "kwa")
	if err := osMkdirAll(currentDir); err != nil {
		t.Fatalf("mkdir current dir: %v", err)
	}

	missingScriptOverride := "missing/measure.sh"
	missingRepoRoot := "missing-repo-root"
	_, err := resolveMeasureScriptPath(currentDir, getenvMap(map[string]string{
		"KWA_MEASURE_SCRIPT": missingScriptOverride,
		"KWA_REPO_ROOT":      missingRepoRoot,
	}))
	if err == nil {
		t.Fatalf("expected not found error, got nil")
	}

	missingScriptPath := filepath.Join(currentDir, missingScriptOverride)
	missingRepoScriptPath := filepath.Join(currentDir, missingRepoRoot, "scripts", "measure.sh")
	if !strings.Contains(err.Error(), "measure script not found; checked") {
		t.Fatalf("error should include checked candidates header, got %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Clean(missingScriptPath)) {
		t.Fatalf("error should include KWA_MEASURE_SCRIPT candidate path, got %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Clean(missingRepoScriptPath)) {
		t.Fatalf("error should include KWA_REPO_ROOT candidate path, got %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(currentDir, "scripts", "measure.sh")) {
		t.Fatalf("error should include upward-search cwd candidate path, got %v", err)
	}
}

func TestRunMeasureScriptWritesStdoutAndStderrToLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")
	markerPath := filepath.Join(dir, "cwd-marker.txt")

	script := "#!/usr/bin/env bash\n" +
		"echo \"line-from-stdout\"\n" +
		"echo \"line-from-stderr\" >&2\n" +
		"touch \"./cwd-marker.txt\"\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	if err := runMeasureScript(context.Background(), scriptPath, nil, logPath); err != nil {
		t.Fatalf("runMeasureScript() error = %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "line-from-stdout") {
		t.Fatalf("expected stdout line in log, got %q", text)
	}
	if !strings.Contains(text, "line-from-stderr") {
		t.Fatalf("expected stderr line in log, got %q", text)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected script cwd marker in repo root, got %v", err)
	}
}

func TestRunMeasureScriptFailsWhenFatalMarkerAppearsInOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")

	script := "#!/usr/bin/env bash\n" +
		"echo \"Final_exception (ConfigurationCheckError): suspend detected\"\n" +
		"exit 0\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := runMeasureScript(context.Background(), scriptPath, nil, logPath)
	if err == nil {
		t.Fatalf("expected fatal output detection error")
	}
	if !strings.Contains(err.Error(), "detected fatal output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMeasureScriptReturnsFatalSummaryOnNonZeroExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")

	script := "#!/usr/bin/env bash\n" +
		"echo \"plain line\"\n" +
		"echo \"Final_exception (ConfigurationCheckError): suspend detected\"\n" +
		"exit 1\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := runMeasureScript(context.Background(), scriptPath, nil, logPath)
	if err == nil {
		t.Fatalf("expected non-zero exit error")
	}
	if !strings.Contains(err.Error(), "see "+logPath) {
		t.Fatalf("error should include log path, got %v", err)
	}
	if !strings.Contains(err.Error(), "exit status 1") {
		t.Fatalf("error should include exit status, got %v", err)
	}
	if !strings.Contains(err.Error(), "Final_exception (ConfigurationCheckError): suspend detected") {
		t.Fatalf("error should include fatal summary line, got %v", err)
	}
}

func TestRunMeasureScriptReturnsGenericErrorSummaryOnNonZeroExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")

	script := "#!/usr/bin/env bash\n" +
		"echo \"something happened\"\n" +
		"echo \"error: could not start container\"\n" +
		"exit 1\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := runMeasureScript(context.Background(), scriptPath, nil, logPath)
	if err == nil {
		t.Fatalf("expected non-zero exit error")
	}
	if !strings.Contains(err.Error(), "error: could not start container") {
		t.Fatalf("error should include generic error summary line, got %v", err)
	}
}

func TestRunMeasureScriptReturnsLastLineSummaryOnNonZeroExit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")

	script := "#!/usr/bin/env bash\n" +
		"echo \"no special markers\"\n" +
		"echo \"final detail\"\n" +
		"exit 1\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := runMeasureScript(context.Background(), scriptPath, nil, logPath)
	if err == nil {
		t.Fatalf("expected non-zero exit error")
	}
	if !strings.Contains(err.Error(), "final detail") {
		t.Fatalf("error should include last line summary, got %v", err)
	}
}

func TestRunMeasureScriptIgnoresHistoricFatalLinesFromPreviousRuns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("Final_exception (ConfigurationCheckError): old run\n"), 0o644); err != nil {
		t.Fatalf("seed historical log: %v", err)
	}

	script := "#!/usr/bin/env bash\n" +
		"echo \"new run okay\"\n" +
		"exit 0\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	if err := runMeasureScript(context.Background(), scriptPath, nil, logPath); err != nil {
		t.Fatalf("runMeasureScript() should ignore old markers, got %v", err)
	}
}

func TestRunMeasureScriptIgnoresHistoricFatalLinesOnNonZeroExitSummary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "scripts", "measure.sh")
	logPath := filepath.Join(dir, "logs", "measure.txt")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("Final_exception (ConfigurationCheckError): old run\n"), 0o644); err != nil {
		t.Fatalf("seed historical log: %v", err)
	}

	script := "#!/usr/bin/env bash\n" +
		"echo \"current run final line\"\n" +
		"exit 1\n"
	if err := writeExecutableScript(scriptPath, script); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := runMeasureScript(context.Background(), scriptPath, nil, logPath)
	if err == nil {
		t.Fatalf("expected non-zero exit error")
	}
	if strings.Contains(err.Error(), "old run") {
		t.Fatalf("error summary should not include historical fatal lines, got %v", err)
	}
	if !strings.Contains(err.Error(), "current run final line") {
		t.Fatalf("error summary should include current run output, got %v", err)
	}
}

// getenvMap creates a deterministic getenv function for resolver tests.
func getenvMap(values map[string]string) func(string) string {
	return func(key string) string {
		if values == nil {
			return ""
		}
		return values[key]
	}
}

// osMkdirAll creates directories used by path-resolution tests.
func osMkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

// writeExecutableScript creates one executable script file and parent directories.
func writeExecutableScript(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o755)
}
