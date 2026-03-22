// Package repository 提供安全模型配置的数据库访问层
// 安全模型配置全局唯一，用于 ShepherdGate 风险检测
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go_lib/core/logging"
)

// SecurityModelConfig 安全模型配置，全局唯一（id=1）
// 用于ShepherdGate风险检测的LLM配置
type SecurityModelConfig struct {
	Provider  string `json:"provider"`
	Endpoint  string `json:"endpoint"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	SecretKey string `json:"secret_key,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// SecurityModelConfigRepository 安全模型配置仓库
type SecurityModelConfigRepository struct {
	db *sql.DB
}

// NewSecurityModelConfigRepository 创建安全模型配置仓库实例
func NewSecurityModelConfigRepository(db *sql.DB) *SecurityModelConfigRepository {
	if db == nil {
		db = GetDB()
	}
	return &SecurityModelConfigRepository{db: db}
}

// CreateSecurityModelConfigTable 创建安全模型配置表
// 使用 IF NOT EXISTS 保证幂等性
func CreateSecurityModelConfigTable(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS security_model_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			provider TEXT,
			endpoint TEXT,
			api_key TEXT,
			model TEXT,
			secret_key TEXT,
			updated_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("failed to create security_model_config table: %w", err)
	}

	// 迁移:为旧版表添加secret_key列
	migrateAddColumn(db, "security_model_config", "secret_key", "TEXT")
	// 迁移:将旧版type列重命名为provider
	migrateRenameColumn(db, "security_model_config", "type", "provider")

	logging.Info("Security model config table created/verified successfully")
	return nil
}

// Save 保存安全模型配置（全局唯一，INSERT OR REPLACE）
func (r *SecurityModelConfigRepository) Save(config *SecurityModelConfig) error {
	if r.db == nil {
		return fmt.Errorf("database not initialized")
	}

	config.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO security_model_config (id, provider, endpoint, api_key, model, secret_key, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`, config.Provider, config.Endpoint, config.APIKey, config.Model, config.SecretKey, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save security model config: %w", err)
	}

	logging.Info("Security model config saved: provider=%s, model=%s", config.Provider, config.Model)
	return nil
}

// Get 获取安全模型配置
// 返回nil表示尚未配置
func (r *SecurityModelConfigRepository) Get() (*SecurityModelConfig, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	row := r.db.QueryRow(`
		SELECT provider, endpoint, api_key, model, secret_key, updated_at
		FROM security_model_config WHERE id = 1
	`)

	var config SecurityModelConfig
	var provider, endpoint, apiKey, model, secretKey, updatedAt sql.NullString

	err := row.Scan(&provider, &endpoint, &apiKey, &model, &secretKey, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query security model config: %w", err)
	}

	config.Provider = provider.String
	config.Endpoint = endpoint.String
	config.APIKey = apiKey.String
	config.Model = model.String
	config.SecretKey = secretKey.String
	config.UpdatedAt = updatedAt.String

	return &config, nil
}
