// Package repository 提供数据库迁移辅助函数
package repository

import (
	"database/sql"
	"fmt"

	"go_lib/core/logging"
)

// migrateAddColumn 安全地为表添加列，忽略已存在的情况
func migrateAddColumn(db *sql.DB, table, column, colType string) {
	// 检查列是否已存在
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == column {
			return // 列已存在
		}
	}

	// 列不存在,执行添加
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
	if err != nil {
		logging.Warning("Failed to add column %s.%s: %v", table, column, err)
	}
}

// migrateRenameColumn 安全地重命名列(SQLite不支持RENAME COLUMN,需通过复制表实现)
func migrateRenameColumn(db *sql.DB, table, oldColumn, newColumn string) {
	// 检查新列是否已存在
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return
	}
	hasOldColumn := false
	hasNewColumn := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == oldColumn {
			hasOldColumn = true
		}
		if name == newColumn {
			hasNewColumn = true
		}
	}
	rows.Close()

	if !hasOldColumn || hasNewColumn {
		return // 旧列不存在或新列已存在,无需迁移
	}

	// 执行数据迁移:添加新列并复制数据
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s TEXT", table, newColumn))
	if err != nil {
		logging.Warning("Failed to add new column %s.%s: %v", table, newColumn, err)
		return
	}

	_, err = db.Exec(fmt.Sprintf("UPDATE %s SET %s = %s", table, newColumn, oldColumn))
	if err != nil {
		logging.Warning("Failed to copy data from %s.%s to %s: %v", table, oldColumn, newColumn, err)
		return
	}

	logging.Info("Successfully migrated column %s.%s -> %s", table, oldColumn, newColumn)
}
