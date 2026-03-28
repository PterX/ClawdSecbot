// Package service provides the database FFI service layer.
// This layer sits between core and repository, handling JSON parsing,
// repository calls, and response formatting.
package service

import (
	"encoding/json"
	"fmt"

	"go_lib/core/logging"
	"go_lib/core/repository"
)

// ========== Database lifecycle management ==========

type InitializeDatabaseRequest struct {
	DBPath          string `json:"db_path"`
	CurrentVersion  string `json:"current_version"`
	VersionFilePath string `json:"version_file_path"`
}

// InitializeDatabase initializes the database connection.
// Request must use JSON input and include
// db_path/current_version/version_file_path.
func InitializeDatabase(requestJSON string) map[string]interface{} {
	var request InitializeDatabaseRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		logging.Error("Failed to parse InitializeDatabase request: %v", err)
		return errorResult(fmt.Errorf("invalid InitializeDatabase request: %w", err))
	}

	if request.DBPath == "" {
		return errorResult(fmt.Errorf("db_path is required"))
	}
	if request.CurrentVersion == "" {
		return errorResult(fmt.Errorf("current_version is required"))
	}
	if request.VersionFilePath == "" {
		return errorResult(fmt.Errorf("version_file_path is required"))
	}

	logging.Info(
		"Initializing database: db_path=%s current_version=%s version_file=%s",
		request.DBPath,
		request.CurrentVersion,
		request.VersionFilePath,
	)

	summary, err := repository.InitDBWithVersion(
		request.DBPath,
		request.CurrentVersion,
		request.VersionFilePath,
	)
	if err != nil {
		logging.Error("Failed to initialize database: %v", err)
		return errorResult(err)
	}

	return successDataResult(map[string]interface{}{
		"path":             request.DBPath,
		"current_version":  summary.CurrentVersion,
		"previous_version": summary.PreviousVersion,
		"version_source":   summary.VersionSource,
		"fresh_install":    summary.FreshInstall,
		"upgraded":         summary.Upgraded,
	})
}

// CloseDatabase closes the database connection.
func CloseDatabase() map[string]interface{} {
	logging.Info("Closing database")

	if err := repository.CloseDB(); err != nil {
		logging.Error("Failed to close database: %v", err)
		return errorResult(err)
	}

	logging.Info("Database closed successfully")
	return map[string]interface{}{
		"success": true,
	}
}
