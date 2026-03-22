package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// setupSecurityModelConfigTestDB 创建临时测试数据库
func setupSecurityModelConfigTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := CreateSecurityModelConfigTable(db); err != nil {
		db.Close()
		t.Fatalf("Failed to create table: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	return db, cleanup
}

// TestSecurityModelConfig_SaveAndGet 验证安全模型配置的保存和读取
func TestSecurityModelConfig_SaveAndGet(t *testing.T) {
	db, cleanup := setupSecurityModelConfigTestDB(t)
	defer cleanup()

	repo := NewSecurityModelConfigRepository(db)

	// 初始状态应返回nil
	config, err := repo.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if config != nil {
		t.Fatal("Expected nil config before save")
	}

	// 保存配置
	err = repo.Save(&SecurityModelConfig{
		Provider: "openai",
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "sk-test-key",
		Model:    "gpt-4",
	})
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// 读取并验证
	config, err = repo.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if config == nil {
		t.Fatal("Expected non-nil config after save")
	}
	if config.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", config.Provider)
	}
	if config.Endpoint != "https://api.openai.com/v1" {
		t.Errorf("Expected endpoint 'https://api.openai.com/v1', got '%s'", config.Endpoint)
	}
	if config.APIKey != "sk-test-key" {
		t.Errorf("Expected api_key 'sk-test-key', got '%s'", config.APIKey)
	}
	if config.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", config.Model)
	}
	if config.UpdatedAt == "" {
		t.Error("Expected updated_at to be set")
	}
}

// TestSecurityModelConfig_Update 验证安全模型配置的更新覆盖
func TestSecurityModelConfig_Update(t *testing.T) {
	db, cleanup := setupSecurityModelConfigTestDB(t)
	defer cleanup()

	repo := NewSecurityModelConfigRepository(db)

	// 保存初始配置
	err := repo.Save(&SecurityModelConfig{
		Provider: "openai",
		APIKey:   "old-key",
		Model:    "gpt-3.5",
	})
	if err != nil {
		t.Fatalf("First save error: %v", err)
	}

	// 更新配置
	err = repo.Save(&SecurityModelConfig{
		Provider: "claude",
		APIKey:   "new-key",
		Model:    "claude-3",
	})
	if err != nil {
		t.Fatalf("Update save error: %v", err)
	}

	// 验证更新后的值
	config, err := repo.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if config.Provider != "claude" {
		t.Errorf("Expected provider 'claude', got '%s'", config.Provider)
	}
	if config.APIKey != "new-key" {
		t.Errorf("Expected api_key 'new-key', got '%s'", config.APIKey)
	}
	if config.Model != "claude-3" {
		t.Errorf("Expected model 'claude-3', got '%s'", config.Model)
	}
}

// TestSecurityModelConfig_WithSecretKey 验证secret_key字段的存储
func TestSecurityModelConfig_WithSecretKey(t *testing.T) {
	db, cleanup := setupSecurityModelConfigTestDB(t)
	defer cleanup()

	repo := NewSecurityModelConfigRepository(db)

	err := repo.Save(&SecurityModelConfig{
		Provider:  "qianfan",
		APIKey:    "access-key",
		Model:     "ernie-bot",
		SecretKey: "secret-key-123",
	})
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	config, err := repo.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if config.SecretKey != "secret-key-123" {
		t.Errorf("Expected secret_key 'secret-key-123', got '%s'", config.SecretKey)
	}
}

// TestSecurityModelConfigRepository_NilDB 验证数据库未初始化时返回错误
func TestSecurityModelConfigRepository_NilDB(t *testing.T) {
	repo := &SecurityModelConfigRepository{db: nil}

	_, err := repo.Get()
	if err == nil {
		t.Error("Expected error for nil db on Get")
	}

	err = repo.Save(&SecurityModelConfig{Provider: "test"})
	if err == nil {
		t.Error("Expected error for nil db on Save")
	}
}

// TestCreateSecurityModelConfigTable_Idempotent 验证建表的幂等性
func TestCreateSecurityModelConfigTable_Idempotent(t *testing.T) {
	db, cleanup := setupSecurityModelConfigTestDB(t)
	defer cleanup()

	// 再次调用建表不应报错
	err := CreateSecurityModelConfigTable(db)
	if err != nil {
		t.Fatalf("Second CreateSecurityModelConfigTable should be idempotent: %v", err)
	}

	// 第三次
	err = CreateSecurityModelConfigTable(db)
	if err != nil {
		t.Fatalf("Third CreateSecurityModelConfigTable should be idempotent: %v", err)
	}
}
