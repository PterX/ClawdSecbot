package service

import (
	"testing"

	"go_lib/core/repository"

	_ "modernc.org/sqlite"
)

// TestInitializeDatabase 验证数据库初始化
func TestInitializeDatabase(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	result := InitializeDatabase(tmpFile)
	defer repository.CloseDB()

	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
	if result["path"] != tmpFile {
		t.Errorf("Expected path=%s, got: %v", tmpFile, result["path"])
	}

	// 验证数据库可用
	db := repository.GetDB()
	if db == nil {
		t.Fatal("GetDB returned nil after InitializeDatabase")
	}
}

// TestInitializeDatabase_InvalidPath 验证无效路径返回错误
func TestInitializeDatabase_InvalidPath(t *testing.T) {
	result := InitializeDatabase("/nonexistent/dir/test.db")
	defer repository.CloseDB()

	if result["success"] != false {
		t.Errorf("Expected success=false for invalid path, got: %v", result)
	}
	if result["error"] == nil {
		t.Error("Expected error message for invalid path")
	}
}

// TestCloseDatabase 验证数据库关闭
func TestCloseDatabase(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	InitializeDatabase(tmpFile)

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
