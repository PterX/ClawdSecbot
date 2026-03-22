package core

import (
	"encoding/json"

	"go_lib/core/logging"
)

// ========== FFI辅助函数（供main包调用）==========

// MarshalJSON 将Go值序列化为JSON字符串
func MarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"success":false,"error":"marshal error"}`
	}
	return string(b)
}

// SuccessResult 成功响应结构
func SuccessResult(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

// ErrorResult 错误响应结构
func ErrorResult(err error) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	}
}

// ========== 全局初始化函数 ==========

// Initialize 初始化全局路径管理器
// workspaceDir: 工作区目录，用于存储日志、数据库、临时文件等
// homeDir: 用户主目录，用于发现 Bot 配置等
func Initialize(workspaceDir, homeDir string) (map[string]interface{}, error) {
	logging.Info("Initializing global path manager: workspaceDir=%s, homeDir=%s", workspaceDir, homeDir)

	pm := GetPathManager()
	if err := pm.Initialize(workspaceDir, homeDir); err != nil {
		logging.Error("Failed to initialize path manager: %v", err)
		return nil, err
	}

	return map[string]interface{}{
		"success":         true,
		"workspace_dir":   workspaceDir,
		"home_dir":        homeDir,
		"log_dir":         pm.GetLogDir(),
		"backup_dir":      pm.GetBackupDir(),
		"policy_dir":      pm.GetPolicyDir(),
		"react_skill_dir": pm.GetReActSkillDir(),
		"db_path":         pm.GetDBPath(),
	}, nil
}

// InitLogging 初始化日志系统
// logDir: 日志目录路径，由 Flutter 传入（可写的 Application Support 目录）。
// 若为空则降级从 PathManager 获取。
func InitLogging(logDir string) (map[string]interface{}, error) {
	if logDir == "" {
		pm := GetPathManager()
		if !pm.IsInitialized() {
			return map[string]interface{}{
				"success": false,
				"error":   "logDir is empty and PathManager not initialized",
			}, nil
		}
		logDir = pm.GetLogDir()
	}

	// 初始化主日志
	if err := logging.InitLogger(logDir, logging.INFO); err != nil {
		return nil, err
	}

	// 初始化历史日志
	if err := logging.InitHistoryLogger(logDir, logging.INFO); err != nil {
		return nil, err
	}

	// 初始化 ShepherdGate 日志
	if err := logging.InitShepherdGateLogger(logDir, logging.INFO); err != nil {
		return nil, err
	}

	logging.Info("Logging system initialized, log dir: %s", logDir)

	return map[string]interface{}{
		"success": true,
		"log_dir": logDir,
	}, nil
}

// GetPaths 获取所有路径信息
func GetPaths() map[string]interface{} {
	pm := GetPathManager()
	if !pm.IsInitialized() {
		return map[string]interface{}{
			"success": false,
			"error":   "PathManager not initialized",
		}
	}

	return map[string]interface{}{
		"success":         true,
		"workspace_dir":   pm.GetWorkspaceDir(),
		"home_dir":        pm.GetHomeDir(),
		"log_dir":         pm.GetLogDir(),
		"backup_dir":      pm.GetBackupDir(),
		"policy_dir":      pm.GetPolicyDir(),
		"react_skill_dir": pm.GetReActSkillDir(),
		"db_path":         pm.GetDBPath(),
	}
}

// ========== 插件管理函数 ==========

// GetRegisteredPlugins 获取所有已注册的插件信息
func GetRegisteredPlugins() map[string]interface{} {
	pm := GetPluginManager()
	infos := pm.GetAllPluginInfos()

	return map[string]interface{}{
		"success": true,
		"data":    infos,
		"count":   len(infos),
	}
}

// ========== 资产扫描函数 ==========

// ScanAllAssets 使用所有插件扫描资产
func ScanAllAssets() (map[string]interface{}, error) {
	logging.Info("Core: Scanning all assets")

	pm := GetPluginManager()
	assets, err := pm.ScanAllAssets()
	if err != nil {
		logging.Error("Core: Scan all assets failed: %v", err)
		return nil, err
	}

	logging.Info("Core: Scan completed, found %d assets", len(assets))
	return map[string]interface{}{
		"success": true,
		"data":    assets,
		"count":   len(assets),
	}, nil
}

// AssessAllRisks 使用所有插件评估风险
func AssessAllRisks(scannedHashes map[string]bool) (map[string]interface{}, error) {
	logging.Info("Core: Assessing all risks")

	pm := GetPluginManager()
	risks, err := pm.AssessAllRisks(scannedHashes)
	if err != nil {
		logging.Error("Core: Assess all risks failed: %v", err)
		return nil, err
	}

	logging.Info("Core: Risk assessment completed, found %d risks", len(risks))
	return map[string]interface{}{
		"success": true,
		"data":    risks,
		"count":   len(risks),
	}, nil
}

// AssessAllRisksFromString 从 JSON 字符串解析 scannedHashes 并评估风险
// JSON 格式: ["hash1", "hash2", ...]
func AssessAllRisksFromString(scannedHashesJSON string) (map[string]interface{}, error) {
	var hashList []string
	if err := json.Unmarshal([]byte(scannedHashesJSON), &hashList); err != nil {
		hashList = nil
	}

	hashSet := make(map[string]bool)
	for _, h := range hashList {
		hashSet[h] = true
	}

	return AssessAllRisks(hashSet)
}

// ========== 风险缓解路由函数 ==========

// MitigateRiskByPlugin routes a mitigation request to the correct plugin via PluginManager.
// riskInfoJSON must contain a "source_plugin" field to identify the target plugin.
func MitigateRiskByPlugin(riskInfoJSON string) string {
	pm := GetPluginManager()
	return pm.MitigateRisk(riskInfoJSON)
}

// ========== 防护控制函数 ==========

// StartProtectionByAsset 启动指定资产实例的防护
// assetName: plugin asset name (e.g. "openclaw"), used to locate the plugin
// assetID: deterministic instance ID from ComputeAssetID, identifies the specific instance
// configJSON: JSON 格式的防护配置字符串
func StartProtectionByAsset(assetName string, assetID string, configJSON string) error {
	var config ProtectionConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return err
	}

	logging.Info("Core: Starting protection for asset: %s (id=%s)", assetName, assetID)

	pm := GetPluginManager()
	if err := pm.StartProtection(assetName, assetID, config); err != nil {
		logging.Error("Core: Start protection failed: %v", err)
		return err
	}

	logging.Info("Core: Protection started for asset: %s (id=%s)", assetName, assetID)
	return nil
}

// StopProtectionByAsset 停止指定资产实例的防护
func StopProtectionByAsset(assetName string, assetID string) error {
	logging.Info("Core: Stopping protection for asset: %s (id=%s)", assetName, assetID)

	pm := GetPluginManager()
	if err := pm.StopProtection(assetName, assetID); err != nil {
		logging.Error("Core: Stop protection failed: %v", err)
		return err
	}

	logging.Info("Core: Protection stopped for asset: %s (id=%s)", assetName, assetID)
	return nil
}

// GetProtectionStatusByAsset 获取指定资产实例的防护状态
func GetProtectionStatusByAsset(assetName string, assetID string) (ProtectionStatus, error) {
	pm := GetPluginManager()
	return pm.GetProtectionStatus(assetName, assetID)
}

// GetAllProtectionStatuses 获取所有资产的防护状态
func GetAllProtectionStatuses() map[string]ProtectionStatus {
	pm := GetPluginManager()
	return pm.GetAllProtectionStatus()
}
