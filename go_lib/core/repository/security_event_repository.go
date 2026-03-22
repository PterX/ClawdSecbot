package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go_lib/core/logging"
)

// SecurityEventRecord 安全事件数据库记录
type SecurityEventRecord struct {
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp"`
	EventType  string `json:"event_type"`
	ActionDesc string `json:"action_desc"`
	RiskType   string `json:"risk_type"`
	Detail     string `json:"detail"`
	Source     string `json:"source"`
}

// SecurityEventRepository 安全事件仓库
type SecurityEventRepository struct {
	db *sql.DB
}

// NewSecurityEventRepository 创建安全事件仓库实例
func NewSecurityEventRepository(db *sql.DB) *SecurityEventRepository {
	if db == nil {
		db = GetDB()
	}
	return &SecurityEventRepository{db: db}
}

// SaveSecurityEventsBatch 批量保存安全事件
func (r *SecurityEventRepository) SaveSecurityEventsBatch(events []*SecurityEventRecord) error {
	if r.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO security_events
		(id, timestamp, event_type, action_desc, risk_type, detail, source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, evt := range events {
		_, err := stmt.Exec(evt.ID, evt.Timestamp, evt.EventType,
			evt.ActionDesc, evt.RiskType, evt.Detail, evt.Source)
		if err != nil {
			logging.Warning("Failed to save security event %s: %v", evt.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

// GetSecurityEvents 获取安全事件列表（按时间倒序）
func (r *SecurityEventRepository) GetSecurityEvents(limit, offset int) ([]*SecurityEventRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(`
		SELECT id, timestamp, event_type, action_desc, risk_type, detail, source
		FROM security_events ORDER BY timestamp DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query security events: %w", err)
	}
	defer rows.Close()

	var events []*SecurityEventRecord
	for rows.Next() {
		evt, err := scanSecurityEvent(rows)
		if err != nil {
			logging.Warning("Failed to scan security event row: %v", err)
			continue
		}
		events = append(events, evt)
	}

	if events == nil {
		events = []*SecurityEventRecord{}
	}
	return events, nil
}

// GetSecurityEventCount 获取安全事件数量
func (r *SecurityEventRepository) GetSecurityEventCount() (int, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM security_events").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count security events: %w", err)
	}
	return count, nil
}

// CleanOldSecurityEvents 清理旧安全事件（保留最近N天）
func (r *SecurityEventRepository) CleanOldSecurityEvents(keepDays int) error {
	if r.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if keepDays <= 0 {
		keepDays = 30
	}

	cutoffTime := time.Now().AddDate(0, 0, -keepDays).UTC().Format(time.RFC3339)
	_, err := r.db.Exec("DELETE FROM security_events WHERE timestamp < ?", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to clean old security events: %w", err)
	}
	return nil
}

// ClearAllSecurityEvents 清空所有安全事件
func (r *SecurityEventRepository) ClearAllSecurityEvents() error {
	if r.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := r.db.Exec("DELETE FROM security_events")
	if err != nil {
		return fmt.Errorf("failed to clear all security events: %w", err)
	}
	return nil
}

func scanSecurityEvent(rows *sql.Rows) (*SecurityEventRecord, error) {
	var evt SecurityEventRecord
	var riskType, detail sql.NullString

	err := rows.Scan(&evt.ID, &evt.Timestamp, &evt.EventType,
		&evt.ActionDesc, &riskType, &detail, &evt.Source)
	if err != nil {
		return nil, err
	}

	evt.RiskType = riskType.String
	evt.Detail = detail.String
	return &evt, nil
}
