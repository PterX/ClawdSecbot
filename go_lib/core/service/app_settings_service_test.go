package service

import (
	"testing"
)

// TestSetLanguage 验证设置语言
func TestSetLanguage(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := SetLanguage("zh")
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
}

// TestGetLanguage 验证获取语言
func TestGetLanguage(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 设置语言
	SetLanguage("zh")

	// 获取语言
	result := GetLanguage()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
	if result["data"] != "zh" {
		t.Errorf("Expected data='zh', got: %v", result["data"])
	}
}

// TestGetLanguage_NotSet 验证未设置时返回空字符串
func TestGetLanguage_NotSet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetLanguage()
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
	if result["data"] != "" {
		t.Errorf("Expected data='', got: %v", result["data"])
	}
}

// TestGetLanguageValue 验证获取语言原始值
func TestGetLanguageValue(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 未设置时返回默认值 "en"
	value := GetLanguageValue()
	if value != "en" {
		t.Errorf("Expected default value 'en', got: %s", value)
	}

	// 设置后返回设置的值
	SetLanguage("zh")
	value = GetLanguageValue()
	if value != "zh" {
		t.Errorf("Expected 'zh', got: %s", value)
	}
}

// TestSaveAppSetting 验证保存通用设置
func TestSaveAppSetting(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	input := `{"key": "theme", "value": "dark"}`
	result := SaveAppSetting(input)
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}

	// 验证保存成功
	check := GetAppSetting("theme")
	if check["success"] != true {
		t.Fatalf("Expected success=true, got: %v", check)
	}
	if check["data"] != "dark" {
		t.Errorf("Expected data='dark', got: %v", check["data"])
	}
}

// TestSaveAppSetting_InvalidJSON 验证JSON解析错误
func TestSaveAppSetting_InvalidJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := SaveAppSetting("bad json")
	if result["success"] != false {
		t.Error("Expected success=false for invalid JSON")
	}
}

// TestSaveAppSetting_MissingKey 验证缺少key字段
func TestSaveAppSetting_MissingKey(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := SaveAppSetting(`{"value": "test"}`)
	if result["success"] != false {
		t.Error("Expected success=false for missing key")
	}
}

// TestGetAppSetting_NotFound 验证获取不存在的设置
func TestGetAppSetting_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := GetAppSetting("nonexistent")
	if result["success"] != true {
		t.Fatalf("Expected success=true, got: %v", result)
	}
	if result["data"] != "" {
		t.Errorf("Expected data='', got: %v", result["data"])
	}
}
