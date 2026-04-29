package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewNamedLoggerWritesAndFiltersByLevel(t *testing.T) {
	logDir := t.TempDir()
	logger, err := NewNamedLogger(logDir, WARNING, "test.log")
	if err != nil {
		t.Fatalf("NewNamedLogger failed: %v", err)
	}
	defer logger.Close()

	logger.Info("hidden message")
	logger.Warning("visible %s", "warning")

	content := readLogFile(t, filepath.Join(logDir, "test.log"))
	if strings.Contains(content, "hidden message") {
		t.Fatalf("Expected INFO message to be filtered, got %q", content)
	}
	if !strings.Contains(content, "[WARNING] visible warning") {
		t.Fatalf("Expected warning message in log, got %q", content)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logDir := t.TempDir()
	logger, err := NewNamedLogger(logDir, ERROR, "test.log")
	if err != nil {
		t.Fatalf("NewNamedLogger failed: %v", err)
	}
	defer logger.Close()

	logger.Warning("hidden warning")
	logger.SetLevel(DEBUG)
	logger.Debug("visible debug")

	content := readLogFile(t, filepath.Join(logDir, "test.log"))
	if strings.Contains(content, "hidden warning") {
		t.Fatalf("Expected warning to be filtered before SetLevel, got %q", content)
	}
	if !strings.Contains(content, "[DEBUG] visible debug") {
		t.Fatalf("Expected debug message after SetLevel, got %q", content)
	}
}

func TestGlobalLoggerLifecycle(t *testing.T) {
	Close()
	if IsInitialized() {
		t.Fatal("Expected logger to start uninitialized")
	}

	logDir := t.TempDir()
	if err := InitLogger(logDir, INFO); err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}
	if !IsInitialized() {
		t.Fatal("Expected logger to be initialized")
	}

	Info("global %s", "message")
	Close()
	if IsInitialized() {
		t.Fatal("Expected Close to clear global logger")
	}

	content := readLogFile(t, filepath.Join(logDir, goLogFileName))
	if !strings.Contains(content, "[INFO] global message") {
		t.Fatalf("Expected global info message, got %q", content)
	}
}

func TestNamedGlobalLoggersWriteExpectedFiles(t *testing.T) {
	CloseHistory()
	CloseShepherdGate()

	logDir := t.TempDir()
	if err := InitHistoryLogger(logDir, INFO); err != nil {
		t.Fatalf("InitHistoryLogger failed: %v", err)
	}
	if err := InitShepherdGateLogger(logDir, DEBUG); err != nil {
		t.Fatalf("InitShepherdGateLogger failed: %v", err)
	}
	defer CloseHistory()
	defer CloseShepherdGate()

	HistoryInfo("history entry")
	ShepherdGateDebug("gate debug")
	ShepherdGateWarning("gate warning")

	historyContent := readLogFile(t, filepath.Join(logDir, goHistoryLogFileName))
	if !strings.Contains(historyContent, "[INFO] history entry") {
		t.Fatalf("Expected history log entry, got %q", historyContent)
	}

	gateContent := readLogFile(t, filepath.Join(logDir, goShepherdGateLogFile))
	if !strings.Contains(gateContent, "[DEBUG] gate debug") || !strings.Contains(gateContent, "[WARNING] gate warning") {
		t.Fatalf("Expected shepherd gate entries, got %q", gateContent)
	}
}

func readLogFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read log file %s: %v", path, err)
	}
	return string(data)
}
