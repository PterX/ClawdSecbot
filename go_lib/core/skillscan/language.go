package skillscan

import (
	"go_lib/core/logging"
	"go_lib/core/repository"
)

// GetLanguageFromAppSettings reads the language setting from app_settings table.
// Exported so that other core packages (e.g. shepherd) can reuse it.
func GetLanguageFromAppSettings() string {
	repo := repository.NewAppSettingsRepository(nil)
	lang, err := repo.GetSetting(repository.SettingKeyLanguage)
	if err != nil {
		logging.Warning("Failed to get language from app_settings: %v", err)
		return ""
	}
	return lang
}
