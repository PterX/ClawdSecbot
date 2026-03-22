package service

import (
	"testing"
)

// TestSaveAuditLog 验证保存审计日志
func TestSaveAuditLog(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	input := `{
		"id": "log-001",
		"timestamp": "2025-01-01T00:00:00Z",
		"request_id": "req-001",
		"model": "gpt-4",
		"request_content": "test request",
		"has_risk": false,
		"action": "ALLOW"
	}`

	result := SaveAuditLog(input)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestSaveAuditLog_InvalidJSON 验证JSON解析错误
func TestSaveAuditLog_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := SaveAuditLog("bad json")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestSaveAuditLogsBatch 验证批量保存审计日志
func TestSaveAuditLogsBatch(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	input := `[
		{
			"id": "log-001",
			"timestamp": "2025-01-01T00:00:00Z",
			"request_id": "req-001",
			"has_risk": false,
			"action": "ALLOW"
		},
		{
			"id": "log-002",
			"timestamp": "2025-01-01T00:01:00Z",
			"request_id": "req-002",
			"has_risk": true,
			"risk_level": "high",
			"action": "BLOCK"
		}
	]`

	result := SaveAuditLogsBatch(input)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestSaveAuditLogsBatch_InvalidJSON 验证JSON解析错误
func TestSaveAuditLogsBatch_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := SaveAuditLogsBatch("bad json")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestGetAuditLogs 验证获取审计日志
func TestGetAuditLogs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	SaveAuditLog(`{
		"id": "log-test",
		"timestamp": "2025-01-01T00:00:00Z",
		"request_id": "req-test",
		"has_risk": false,
		"action": "ALLOW"
	}`)

	result := GetAuditLogs(`{"limit": 10, "offset": 0}`)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestGetAuditLogs_InvalidJSON 验证JSON解析错误
func TestGetAuditLogs_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetAuditLogs("bad")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestGetAuditLogCount 验证获取审计日志数量
func TestGetAuditLogCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetAuditLogCount(`{"risk_only": false}`)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestGetAuditLogCount_InvalidJSON 验证JSON解析错误
func TestGetAuditLogCount_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetAuditLogCount("bad")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestGetAuditLogStatistics 验证获取审计日志统计
func TestGetAuditLogStatistics(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetAuditLogStatistics()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestCleanOldAuditLogs 验证清理旧审计日志
func TestCleanOldAuditLogs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := CleanOldAuditLogs(`{"keep_days": 30}`)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestCleanOldAuditLogs_InvalidJSON 验证JSON解析错误
func TestCleanOldAuditLogs_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := CleanOldAuditLogs("bad")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestClearAllAuditLogs 验证清空所有审计日志
func TestClearAllAuditLogs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := ClearAllAuditLogs()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}
