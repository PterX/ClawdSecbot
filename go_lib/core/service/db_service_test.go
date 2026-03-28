package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go_lib/core/repository"

	_ "modernc.org/sqlite"
)

// TestInitializeDatabase 验证数据库初始化
func TestInitializeDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.db")
	versionFile := filepath.Join(tmpDir, "bot_sec_manager.version")

	result := InitializeDatabase(mustInitDatabaseRequestJSON(t, tmpFile, versionFile, "1.0.1"))
	defer repository.CloseDB()

	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data payload, got: %v", result)
	}
	if data["path"] != tmpFile {
		t.Errorf("Expected path=%s, got: %v", tmpFile, data["path"])
	}

	// 验证数据库可用
	db := repository.GetDB()
	if db == nil {
		t.Fatal("GetDB returned nil after InitializeDatabase")
	}

	content, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("Expected version file to be written: %v", err)
	}
	if string(content) != "1.0.1\n" {
		t.Fatalf("Expected version file content 1.0.1, got %q", string(content))
	}
}

// TestInitializeDatabase_InvalidPath 验证无效路径返回错误
func TestInitializeDatabase_InvalidPath(t *testing.T) {
	result := InitializeDatabase(mustInitDatabaseRequestJSON(
		t,
		"/nonexistent/dir/test.db",
		"/nonexistent/dir/bot_sec_manager.version",
		"1.0.1",
	))
	defer repository.CloseDB()

	if result["success"] != false {
		t.Errorf("Expected success=false for invalid path, got: %v", result)
	}
	if result["error"] == nil {
		t.Error("Expected error message for invalid path")
	}
}

func TestInitializeDatabase_InvalidRequest(t *testing.T) {
	result := InitializeDatabase(`{"db_path":""}`)
	if result["success"] != false {
		t.Fatalf("Expected success=false, got: %v", result)
	}
	if result["error"] == nil {
		t.Fatal("Expected validation error for invalid request")
	}
}

// TestCloseDatabase 验证数据库关闭
func TestCloseDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.db")
	versionFile := filepath.Join(tmpDir, "bot_sec_manager.version")
	InitializeDatabase(mustInitDatabaseRequestJSON(t, tmpFile, versionFile, "1.0.1"))

	result := CloseDatabase()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestCloseDatabase_NotInitialized 验证未初始化时关闭不报错
func TestCloseDatabase_NotInitialized(t *testing.T) {
	result := CloseDatabase()
	if result["success"] != true {
		t.Errorf("Expected success=true for closing uninitialized DB, got: %v", result)
	}
}

func mustInitDatabaseRequestJSON(
	t *testing.T,
	dbPath,
	versionFilePath,
	currentVersion string,
) string {
	t.Helper()

	request := InitializeDatabaseRequest{
		DBPath:          dbPath,
		CurrentVersion:  currentVersion,
		VersionFilePath: versionFilePath,
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal test request: %v", err)
	}

	return string(data)
}
