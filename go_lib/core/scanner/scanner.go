// Package scanner 提供通用的资产扫描服务
// 封装了系统信息采集、规则加载和资产检测的完整流程，
// 供各Bot插件复用，避免重复实现扫描逻辑
package scanner

import (
	"go_lib/core"
	"go_lib/core/logging"
)

// AssetScanner 资产扫描服务
// 组合了检测引擎和系统采集器，提供完整的资产扫描流程
type AssetScanner struct {
	// engine 资产检测引擎，负责规则匹配
	engine *core.AssetDetectionEngine
	// collector 系统信息采集器，负责获取系统快照
	collector core.Collector
}

// NewAssetScanner 创建资产扫描服务实例
// collector 为系统信息采集器，可传入 core.NewCollector() 创建的平台采集器或测试用的模拟采集器
func NewAssetScanner(collector core.Collector) *AssetScanner {
	return &AssetScanner{
		engine:    core.NewEngine(),
		collector: collector,
	}
}

// LoadRules 批量加载检测规则到引擎中
func (s *AssetScanner) LoadRules(rules []core.AssetFinderRule) {
	for _, rule := range rules {
		s.engine.LoadRule(rule)
	}
}

// Scan 执行资产扫描，返回检测到的资产列表
// 流程：采集系统快照 → 规则匹配 → 返回匹配的资产
func (s *AssetScanner) Scan() ([]core.Asset, error) {
	logging.Info("Asset scanner starting scan...")

	// 1. 采集系统快照
	snapshot, err := s.collector.Collect()
	if err != nil {
		logging.Error("Failed to collect system snapshot: %v", err)
		return nil, err
	}

	// 2. 执行规则检测
	assets, err := s.engine.Detect(snapshot)
	if err != nil {
		logging.Error("Asset detection failed: %v", err)
		return nil, err
	}

	if assets == nil {
		assets = []core.Asset{}
	}

	logging.Info("Asset scanner completed, found %d asset(s)", len(assets))
	return assets, nil
}

// MergeAssetsByName 按资产名称合并同类资产
// 将多条检测规则匹配到的同一类资产合并为一个，合并规则：
//   - 端口列表：去重合并
//   - 进程路径：去重合并
//   - 元数据：不覆盖已有值
//   - 服务名称：取第一个非空值
func MergeAssetsByName(assets []core.Asset, targetName string, targetType string) *core.Asset {
	if len(assets) == 0 {
		return nil
	}

	merged := &core.Asset{
		Name:         targetName,
		Type:         targetType,
		Ports:        make([]int, 0),
		ProcessPaths: make([]string, 0),
		Metadata:     make(map[string]string),
	}

	for _, asset := range assets {
		// 合并端口（去重）
		for _, port := range asset.Ports {
			if !containsInt(merged.Ports, port) {
				merged.Ports = append(merged.Ports, port)
			}
		}

		// 合并进程路径（去重）
		for _, path := range asset.ProcessPaths {
			if !containsString(merged.ProcessPaths, path) {
				merged.ProcessPaths = append(merged.ProcessPaths, path)
			}
		}

		// 合并元数据（不覆盖已有值）
		for k, v := range asset.Metadata {
			if merged.Metadata[k] == "" {
				merged.Metadata[k] = v
			}
		}

		// 补充版本号（取第一个非空值）
		if merged.Version == "" && asset.Version != "" {
			merged.Version = asset.Version
		}

		// 补充服务名称（取第一个非空值）
		if merged.ServiceName == "" && asset.ServiceName != "" {
			merged.ServiceName = asset.ServiceName
		}
	}

	return merged
}

// containsInt 检查int切片中是否包含指定值
func containsInt(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// containsString 检查字符串切片中是否包含指定值
func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
