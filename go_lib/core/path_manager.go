package core

import (
	"os"
	"path/filepath"
	"sync"

	"go_lib/core/logging"
)

// PathManager 全局路径管理器
// 负责管理所有路径，由Flutter初始化时传入基础路径
// 其他路径由基础路径派生，确保路径管理的统一性
type PathManager struct {
	mu sync.RWMutex

	initialized bool

	// 基础路径（由Flutter传入）
	workspaceDir string // 工作区目录，用于存储日志、数据库、临时文件等
	homeDir      string // 用户主目录（如 /Users/username）

	// 派生路径
	logDir        string // 日志目录：{workspaceDir}/logs
	backupDir     string // 备份目录：{workspaceDir}/backups
	policyDir     string // 策略目录：{homeDir}/.botsec/policies（用于sandbox-exec）
	reactSkillDir string // ReAct 风险技能目录：{workspaceDir}/skills/shepherd_gate
	scanSkillDir  string // 扫描技能目录：{workspaceDir}/skills/skill_scanner
	dbPath        string // 数据库路径：{workspaceDir}/botsec.db
}

var (
	globalPathManager *PathManager
	pathManagerOnce   sync.Once
)

// GetPathManager 获取全局路径管理器实例
func GetPathManager() *PathManager {
	pathManagerOnce.Do(func() {
		globalPathManager = &PathManager{}
	})
	return globalPathManager
}

// Initialize 初始化路径管理器
// workspaceDir: 工作区目录（由Flutter获取并传入）
// homeDir: 用户主目录
func (pm *PathManager) Initialize(workspaceDir, homeDir string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.initialized {
		logging.Warning("PathManager already initialized, skipping")
		return nil
	}

	pm.workspaceDir = workspaceDir
	pm.homeDir = homeDir

	// 派生其他路径
	pm.logDir = filepath.Join(workspaceDir, "logs")
	pm.backupDir = filepath.Join(workspaceDir, "backups")
	pm.policyDir = filepath.Join(homeDir, ".botsec", "policies")
	pm.reactSkillDir = filepath.Join(workspaceDir, "skills", "shepherd_gate")
	pm.scanSkillDir = filepath.Join(workspaceDir, "skills", "skill_scanner")
	pm.dbPath = filepath.Join(workspaceDir, "botsec.db")

	// 确保必要的目录存在
	if err := pm.ensureDirectories(); err != nil {
		logging.Error("Failed to ensure directories: %v", err)
		return err
	}

	pm.initialized = true
	logging.Info("PathManager initialized: workspaceDir=%s, homeDir=%s", workspaceDir, homeDir)
	logging.Info("Derived paths: logDir=%s, backupDir=%s, policyDir=%s, dbPath=%s",
		pm.logDir, pm.backupDir, pm.policyDir, pm.dbPath)
	logging.Info("Derived paths (skills): reactSkillDir=%s, scanSkillDir=%s", pm.reactSkillDir, pm.scanSkillDir)

	return nil
}

// ensureDirectories 确保必要的目录存在
func (pm *PathManager) ensureDirectories() error {
	dirs := []string{pm.logDir, pm.backupDir, pm.policyDir, pm.reactSkillDir, pm.scanSkillDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// IsInitialized 检查是否已初始化
func (pm *PathManager) IsInitialized() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.initialized
}

// ========== Getter方法 ==========

// GetWorkspaceDir 获取工作区目录
func (pm *PathManager) GetWorkspaceDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.workspaceDir
}

// GetHomeDir 获取用户主目录
func (pm *PathManager) GetHomeDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.homeDir
}

// GetLogDir 获取日志目录
func (pm *PathManager) GetLogDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.logDir
}

// GetBackupDir 获取备份目录
func (pm *PathManager) GetBackupDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.backupDir
}

// GetPolicyDir 获取策略目录（用于sandbox-exec）
func (pm *PathManager) GetPolicyDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.policyDir
}

// GetReActSkillDir 获取 ReAct 风险技能目录
func (pm *PathManager) GetReActSkillDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.reactSkillDir
}

// GetScanSkillDir 获取扫描技能目录
func (pm *PathManager) GetScanSkillDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.scanSkillDir
}

// GetDBPath 获取数据库文件路径
func (pm *PathManager) GetDBPath() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.dbPath
}

// ========== 路径构建辅助方法 ==========

// JoinWorkspace 在工作区目录下拼接路径
func (pm *PathManager) JoinWorkspace(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.workspaceDir}, elem...)
	return filepath.Join(parts...)
}

// JoinHome 在用户主目录下拼接路径
func (pm *PathManager) JoinHome(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.homeDir}, elem...)
	return filepath.Join(parts...)
}

// JoinLog 在日志目录下拼接路径
func (pm *PathManager) JoinLog(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.logDir}, elem...)
	return filepath.Join(parts...)
}

// JoinBackup 在备份目录下拼接路径
func (pm *PathManager) JoinBackup(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.backupDir}, elem...)
	return filepath.Join(parts...)
}

// JoinPolicy 在策略目录下拼接路径
func (pm *PathManager) JoinPolicy(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.policyDir}, elem...)
	return filepath.Join(parts...)
}

// JoinReActSkill 在 ReAct 风险技能目录下拼接路径
func (pm *PathManager) JoinReActSkill(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.reactSkillDir}, elem...)
	return filepath.Join(parts...)
}

// JoinScanSkill 在扫描技能目录下拼接路径
func (pm *PathManager) JoinScanSkill(elem ...string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	parts := append([]string{pm.scanSkillDir}, elem...)
	return filepath.Join(parts...)
}
