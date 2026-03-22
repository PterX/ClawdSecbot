// Package service 提供安全模型配置的业务服务层
package service

import (
	"encoding/json"

	"go_lib/core/logging"
	"go_lib/core/repository"
)

// SaveSecurityModelConfig 保存安全模型配置
func SaveSecurityModelConfig(jsonStr string) map[string]interface{} {
	var config repository.SecurityModelConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		logging.Error("Failed to parse security model config JSON: %v", err)
		return errorMessageResult("invalid JSON: " + err.Error())
	}

	repo := repository.NewSecurityModelConfigRepository(nil)
	if err := repo.Save(&config); err != nil {
		logging.Error("Failed to save security model config: %v", err)
		return errorResult(err)
	}

	return successResult()
}

// GetSecurityModelConfig 获取安全模型配置
func GetSecurityModelConfig() map[string]interface{} {
	repo := repository.NewSecurityModelConfigRepository(nil)
	config, err := repo.Get()
	if err != nil {
		logging.Error("Failed to get security model config: %v", err)
		return errorResult(err)
	}

	if config == nil {
		return successDataResult(nil)
	}
	return successDataResult(config)
}
