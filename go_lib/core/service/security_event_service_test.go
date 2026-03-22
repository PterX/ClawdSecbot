package service

import (
	"encoding/json"
	"testing"
)

// TestSaveSecurityEventsBatch 验证批量保存安全事件
func TestSaveSecurityEventsBatch(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	input := `[
		{
			"id": "sevt_1_1",
			"timestamp": "2026-03-15T10:00:00Z",
			"event_type": "tool_execution",
			"action_desc": "执行ls命令列出文件",
			"risk_type": "",
			"detail": "",
			"source": "react_agent"
		},
		{
			"id": "sevt_2_2",
			"timestamp": "2026-03-15T10:01:00Z",
			"event_type": "blocked",
			"action_desc": "尝试rm -rf /删除系统文件",
			"risk_type": "权限提升",
			"detail": "致命命令检测",
			"source": "heuristic"
		}
	]`

	result := SaveSecurityEventsBatch(input)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestSaveSecurityEventsBatch_InvalidJSON 验证JSON解析错误
func TestSaveSecurityEventsBatch_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := SaveSecurityEventsBatch("bad json")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestGetSecurityEvents 验证获取安全事件
func TestGetSecurityEvents(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 先保存一条
	SaveSecurityEventsBatch(`[{
		"id": "sevt_test_1",
		"timestamp": "2026-03-15T10:00:00Z",
		"event_type": "tool_execution",
		"action_desc": "读取配置文件",
		"source": "react_agent"
	}]`)

	result := GetSecurityEvents(`{"limit": 10, "offset": 0}`)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}

	data := result["data"]
	events, ok := data.([]interface{})
	if !ok {
		// 可能是 JSON 格式，尝试 marshal/unmarshal
		b, _ := json.Marshal(data)
		var arr []interface{}
		json.Unmarshal(b, &arr)
		events = arr
	}
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
}

// TestGetSecurityEvents_InvalidJSON 验证JSON解析错误
func TestGetSecurityEvents_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetSecurityEvents("bad")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestGetSecurityEventCount 验证获取安全事件数量
func TestGetSecurityEventCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetSecurityEventCount()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestClearAllSecurityEvents 验证清空安全事件
func TestClearAllSecurityEvents(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := ClearAllSecurityEvents()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}
