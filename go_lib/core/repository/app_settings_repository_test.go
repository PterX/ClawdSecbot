package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// setupAppSettingsTestDB 创建临时测试数据库，返回数据库连接和清理函数
func setupAppSettingsTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 创建应用设置表
	if err := CreateAppSettingsTable(db); err != nil {
		db.Close()
		t.Fatalf("Failed to create app_settings table: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

// TestSaveAndGetSetting 验证设置项的保存和读取
func TestSaveAndGetSetting(t *testing.T) {
	db, cleanup := setupAppSettingsTestDB(t)
	defer cleanup()

	repo := NewAppSettingsRepository(db)

	// 初始状态应返回空字符串
	value, err := repo.GetSetting("language")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string before save, got '%s'", value)
	}

	// 保存设置
	err = repo.SaveSetting("language", "zh")
	if err != nil {
		t.Fatalf("SaveSetting error: %v", err)
	}

	// 读取并验证
	value, err = repo.GetSetting("language")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if value != "zh" {
		t.Errorf("Expected 'zh', got '%s'", value)
	}
}

// TestSetting_Update 验证设置项的更新覆盖
func TestSetting_Update(t *testing.T) {
	db, cleanup := setupAppSettingsTestDB(t)
	defer cleanup()

	repo := NewAppSettingsRepository(db)

	// 保存初始值
	err := repo.SaveSetting("language", "en")
	if err != nil {
		t.Fatalf("SaveSetting error: %v", err)
	}

	// 更新值
	err = repo.SaveSetting("language", "zh")
	if err != nil {
		t.Fatalf("SaveSetting update error: %v", err)
	}

	// 验证更新后的值
	value, err := repo.GetSetting("language")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if value != "zh" {
		t.Errorf("Expected 'zh' after update, got '%s'", value)
	}
}

// TestDeleteSetting 验证设置项的删除
func TestDeleteSetting(t *testing.T) {
	db, cleanup := setupAppSettingsTestDB(t)
	defer cleanup()

	repo := NewAppSettingsRepository(db)

	// 保存设置
	err := repo.SaveSetting("language", "zh")
	if err != nil {
		t.Fatalf("SaveSetting error: %v", err)
	}

	// 删除设置
	err = repo.DeleteSetting("language")
	if err != nil {
		t.Fatalf("DeleteSetting error: %v", err)
	}

	// 验证删除后返回空字符串
	value, err := repo.GetSetting("language")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string after delete, got '%s'", value)
	}
}

// TestMultipleSettings 验证多个设置项的独立存储
func TestMultipleSettings(t *testing.T) {
	db, cleanup := setupAppSettingsTestDB(t)
	defer cleanup()

	repo := NewAppSettingsRepository(db)

	// 保存多个设置
	err := repo.SaveSetting("language", "zh")
	if err != nil {
		t.Fatalf("SaveSetting error: %v", err)
	}
	err = repo.SaveSetting("theme", "dark")
	if err != nil {
		t.Fatalf("SaveSetting error: %v", err)
	}

	// 验证各自独立
	lang, err := repo.GetSetting("language")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if lang != "zh" {
		t.Errorf("Expected language 'zh', got '%s'", lang)
	}

	theme, err := repo.GetSetting("theme")
	if err != nil {
		t.Fatalf("GetSetting error: %v", err)
	}
	if theme != "dark" {
		t.Errorf("Expected theme 'dark', got '%s'", theme)
	}
}

// TestSettingKeyLanguage 验证语言键常量
func TestSettingKeyLanguage(t *testing.T) {
	if SettingKeyLanguage != "language" {
		t.Errorf("Expected SettingKeyLanguage to be 'language', got '%s'", SettingKeyLanguage)
	}
}

// TestSettingKeyIsFirstLaunch 验证首次启动键常量
func TestSettingKeyIsFirstLaunch(t *testing.T) {
	if SettingKeyIsFirstLaunch != "is_first_launch" {
		t.Errorf("Expected SettingKeyIsFirstLaunch to be 'is_first_launch', got '%s'", SettingKeyIsFirstLaunch)
	}
}
