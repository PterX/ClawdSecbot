// Package service 提供数据库FFI业务逻辑层
// 该层位于core和repository之间，负责JSON解析、调用repository方法、格式化响应
// 所有函数返回 map[string]interface{}，由插件层的CGo导出函数调用
package service

import (
	"go_lib/core/logging"
	"go_lib/core/repository"
)

// ========== 数据库生命周期管理 ==========

// InitializeDatabase 初始化数据库连接
// dbPath 为SQLite数据库文件的完整路径
func InitializeDatabase(dbPath string) map[string]interface{} {
	logging.Info("Initializing database: %s", dbPath)

	if err := repository.InitDB(dbPath); err != nil {
		logging.Error("Failed to initialize database: %v", err)
		return errorResult(err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    dbPath,
	}
}

// CloseDatabase 关闭数据库连接
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
