package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go_lib/core/logging"
)

// ApiMetrics API调用指标记录
type ApiMetrics struct {
	ID               int    `json:"id,omitempty"`
	Timestamp        string `json:"timestamp"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	ToolCallCount    int    `json:"tool_call_count"`
	Model            string `json:"model,omitempty"`
	IsBlocked        bool   `json:"is_blocked"`
	RiskLevel        string `json:"risk_level,omitempty"`
	AssetName        string `json:"asset_name,omitempty"`
}

// TokenTrendPoint Token趋势数据点
type TokenTrendPoint struct {
	Timestamp        string `json:"timestamp"`
	Tokens           int    `json:"tokens"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
}

// ToolCallTrendPoint 工具调用趋势数据点
type ToolCallTrendPoint struct {
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
}

// ApiStatistics API统计数据
type ApiStatistics struct {
	TotalTokens           int                  `json:"total_tokens"`
	TotalPromptTokens     int                  `json:"total_prompt_tokens"`
	TotalCompletionTokens int                  `json:"total_completion_tokens"`
	TotalToolCalls        int                  `json:"total_tool_calls"`
	RequestCount          int                  `json:"request_count"`
	BlockedCount          int                  `json:"blocked_count"`
	TokenTrend            []*TokenTrendPoint   `json:"token_trend"`
	ToolCallTrend         []*ToolCallTrendPoint `json:"tool_call_trend"`
}

// MetricsRepository API指标仓库
type MetricsRepository struct {
	db *sql.DB
}

// NewMetricsRepository 创建API指标仓库实例
func NewMetricsRepository(db *sql.DB) *MetricsRepository {
	if db == nil {
		db = GetDB()
	}
	return &MetricsRepository{db: db}
}

// SaveApiMetrics 保存API指标
func (r *MetricsRepository) SaveApiMetrics(metrics *ApiMetrics) error {
	if r.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if metrics.Timestamp == "" {
		metrics.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	isBlocked := 0
	if metrics.IsBlocked {
		isBlocked = 1
	}

	_, err := r.db.Exec(`
		INSERT INTO api_metrics (timestamp, prompt_tokens, completion_tokens, total_tokens, 
			tool_call_count, model, is_blocked, risk_level, asset_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, metrics.Timestamp, metrics.PromptTokens, metrics.CompletionTokens, metrics.TotalTokens,
		metrics.ToolCallCount, metrics.Model, isBlocked, metrics.RiskLevel, metrics.AssetName)
	if err != nil {
		return fmt.Errorf("failed to save api metrics: %w", err)
	}

	return nil
}

// GetApiStatistics 获取API统计数据
func (r *MetricsRepository) GetApiStatistics(durationSeconds int, assetName string) (*ApiStatistics, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if durationSeconds <= 0 {
		durationSeconds = 86400 // 默认24小时
	}

	cutoffTime := time.Now().Add(-time.Duration(durationSeconds) * time.Second).UTC().Format(time.RFC3339)

	whereClause := "WHERE timestamp >= ?"
	params := []interface{}{cutoffTime}

	if assetName != "" {
		whereClause += " AND asset_name = ?"
		params = append(params, assetName)
	}

	// 汇总统计
	row := r.db.QueryRow(fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(tool_call_count), 0),
			COUNT(*),
			COALESCE(SUM(is_blocked), 0)
		FROM api_metrics %s
	`, whereClause), params...)

	var stats ApiStatistics
	err := row.Scan(&stats.TotalTokens, &stats.TotalPromptTokens, &stats.TotalCompletionTokens,
		&stats.TotalToolCalls, &stats.RequestCount, &stats.BlockedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get api statistics: %w", err)
	}

	// Token趋势
	tokenTrend, err := r.getTokenTrend(cutoffTime, assetName)
	if err != nil {
		logging.Warning("Failed to get token trend: %v", err)
	}
	stats.TokenTrend = tokenTrend

	// 工具调用趋势
	toolCallTrend, err := r.getToolCallTrend(cutoffTime, assetName)
	if err != nil {
		logging.Warning("Failed to get tool call trend: %v", err)
	}
	stats.ToolCallTrend = toolCallTrend

	if stats.TokenTrend == nil {
		stats.TokenTrend = []*TokenTrendPoint{}
	}
	if stats.ToolCallTrend == nil {
		stats.ToolCallTrend = []*ToolCallTrendPoint{}
	}

	return &stats, nil
}

// getTokenTrend 获取Token使用趋势（按分钟聚合）
func (r *MetricsRepository) getTokenTrend(cutoffTime, assetName string) ([]*TokenTrendPoint, error) {
	whereClause := "WHERE timestamp >= ?"
	params := []interface{}{cutoffTime}

	if assetName != "" {
		whereClause += " AND asset_name = ?"
		params = append(params, assetName)
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT 
			strftime('%%Y-%%m-%%d %%H:%%M:00', timestamp) as minute,
			SUM(total_tokens) as tokens,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
		FROM api_metrics %s
		GROUP BY minute ORDER BY minute ASC LIMIT 60
	`, whereClause), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []*TokenTrendPoint
	for rows.Next() {
		var point TokenTrendPoint
		if err := rows.Scan(&point.Timestamp, &point.Tokens, &point.PromptTokens, &point.CompletionTokens); err != nil {
			continue
		}
		trend = append(trend, &point)
	}

	return trend, nil
}

// getToolCallTrend 获取工具调用趋势（按分钟聚合）
func (r *MetricsRepository) getToolCallTrend(cutoffTime, assetName string) ([]*ToolCallTrendPoint, error) {
	whereClause := "WHERE timestamp >= ?"
	params := []interface{}{cutoffTime}

	if assetName != "" {
		whereClause += " AND asset_name = ?"
		params = append(params, assetName)
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT 
			strftime('%%Y-%%m-%%d %%H:%%M:00', timestamp) as minute,
			SUM(tool_call_count) as count
		FROM api_metrics %s
		GROUP BY minute ORDER BY minute ASC LIMIT 60
	`, whereClause), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []*ToolCallTrendPoint
	for rows.Next() {
		var point ToolCallTrendPoint
		if err := rows.Scan(&point.Timestamp, &point.Count); err != nil {
			continue
		}
		trend = append(trend, &point)
	}

	return trend, nil
}

// GetRecentApiMetrics 获取最近的API指标记录
func (r *MetricsRepository) GetRecentApiMetrics(limit int) ([]*ApiMetrics, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(`SELECT id, timestamp, prompt_tokens, completion_tokens, total_tokens,
		tool_call_count, model, is_blocked, risk_level, asset_name
		FROM api_metrics ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent api metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*ApiMetrics
	for rows.Next() {
		var m ApiMetrics
		var isBlocked int
		var model, riskLevel, assetName sql.NullString

		if err := rows.Scan(&m.ID, &m.Timestamp, &m.PromptTokens, &m.CompletionTokens,
			&m.TotalTokens, &m.ToolCallCount, &model, &isBlocked, &riskLevel, &assetName); err != nil {
			continue
		}

		m.IsBlocked = isBlocked == 1
		m.Model = model.String
		m.RiskLevel = riskLevel.String
		m.AssetName = assetName.String
		metrics = append(metrics, &m)
	}

	if metrics == nil {
		metrics = []*ApiMetrics{}
	}
	return metrics, nil
}

// CleanOldApiMetrics 清理旧API指标（保留最近N天）
func (r *MetricsRepository) CleanOldApiMetrics(keepDays int) error {
	if r.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if keepDays <= 0 {
		keepDays = 7
	}

	cutoffTime := time.Now().AddDate(0, 0, -keepDays).UTC().Format(time.RFC3339)
	_, err := r.db.Exec("DELETE FROM api_metrics WHERE timestamp < ?", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to clean old api metrics: %w", err)
	}

	return nil
}

// GetDailyTokenUsage 获取指定资产当日的Token使用量
func (r *MetricsRepository) GetDailyTokenUsage(assetName string) (int, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startStr := startOfDay.UTC().Format(time.RFC3339)

	var dailyTokens int
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(total_tokens), 0)
		FROM api_metrics WHERE asset_name = ? AND timestamp >= ?
	`, assetName, startStr).Scan(&dailyTokens)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily token usage: %w", err)
	}

	return dailyTokens, nil
}
