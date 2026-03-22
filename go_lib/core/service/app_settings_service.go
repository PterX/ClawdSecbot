package service

import (
	"encoding/json"

	"go_lib/core/logging"
	"go_lib/core/repository"
)

// ========== 应用设置操作 ==========

// SaveAppSetting 保存应用设置
// jsonStr 格式: {"key": "xxx", "value": "xxx"}
func SaveAppSetting(jsonStr string) map[string]interface{} {
	var input struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
		logging.Error("Failed to parse app setting JSON: %v", err)
		return errorMessageResult("invalid JSON: " + err.Error())
	}

	if input.Key == "" {
		return errorMessageResult("key is required")
	}

	repo := repository.NewAppSettingsRepository(nil)
	if err := repo.SaveSetting(input.Key, input.Value); err != nil {
		logging.Error("Failed to save app setting: %v", err)
		return errorResult(err)
	}

	return successResult()
}

// GetAppSetting 获取应用设置
func GetAppSetting(key string) map[string]interface{} {
	repo := repository.NewAppSettingsRepository(nil)
	value, err := repo.GetSetting(key)
	if err != nil {
		logging.Error("Failed to get app setting: %v", err)
		return errorResult(err)
	}

	return successDataResult(value)
}

// SetLanguage 设置语言
// 这是一个便捷方法，直接设置 language 键
// 注意：此方法只保存到数据库，不会更新运行时的 ShepherdGate/SkillAgent
// 运行时更新应由 FFI 层调用 openclaw.UpdateLanguage() 完成
func SetLanguage(lang string) map[string]interface{} {
	repo := repository.NewAppSettingsRepository(nil)
	if err := repo.SaveSetting(repository.SettingKeyLanguage, lang); err != nil {
		logging.Error("Failed to set language: %v", err)
		return errorResult(err)
	}

	logging.Info("Language setting saved: %s", lang)
	return successResult()
}

// GetLanguage 获取语言设置
// 如果未设置，返回空字符串
func GetLanguage() map[string]interface{} {
	repo := repository.NewAppSettingsRepository(nil)
	value, err := repo.GetSetting(repository.SettingKeyLanguage)
	if err != nil {
		logging.Error("Failed to get language: %v", err)
		return errorResult(err)
	}

	return successDataResult(value)
}

// GetLanguageValue 获取语言设置的原始值（供内部使用）
// 如果未设置，返回默认值 "en"
func GetLanguageValue() string {
	repo := repository.NewAppSettingsRepository(nil)
	value, err := repo.GetSetting(repository.SettingKeyLanguage)
	if err != nil {
		logging.Warning("Failed to get language, using default: %v", err)
		return "en"
	}
	if value == "" {
		return "en"
	}
	return value
}
