package service

import (
	"testing"

	"go_lib/core/repository"

	_ "modernc.org/sqlite"
)

// setupTestDB 初始化内存数据库供测试使用
// 会创建所有表结构，返回清理函数
func setupTestDB(t *testing.T) func() {
	t.Helper()
	if err := repository.InitDB(":memory:"); err != nil {
		t.Fatalf("Failed to init test database: %v", err)
	}
	return func() {
		repository.CloseDB()
	}
}
