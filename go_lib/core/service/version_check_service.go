package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go_lib/core/callback_bridge"
	"go_lib/core/logging"
)

// 版本检查服务配置常量
const (
	defaultCheckURL    = "https://xxxxx/version"
	defaultAESKey      = ""
	initialCheckDelay  = 60 * time.Second // 首次检查延迟
	periodicCheckDelay = 4 * time.Hour    // 周期检查间隔
	maxRetryDelay      = 1 * time.Hour    // 最大重试延迟
	httpTimeout        = 10 * time.Second // HTTP 超时
)

// VersionInfo 版本信息结构
type VersionInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Hash        string `json:"hash"`
	ForceUpdate bool   `json:"force_update"`
	ChangeLog   string `json:"change_log"`
}

// VersionCheckConfig 版本检查服务配置
type VersionCheckConfig struct {
	CurrentVersion string `json:"current_version"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Language       string `json:"language"`
	Enabled        bool   `json:"enabled"`
}

// VersionUpdateSender 版本更新发送器接口（用于测试）
type VersionUpdateSender interface {
	IsRunning() bool
	SendVersionUpdate(payload map[string]interface{})
}

// VersionCheckService 版本检查服务
type VersionCheckService struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 配置
	checkURL   string
	aesKey     []byte
	currentVer string
	os         string
	arch       string
	language   string
	enabled    bool

	// 回调桥接器
	bridge VersionUpdateSender

	// 重试控制
	retryCount int
	checkTimer *time.Timer

	mu sync.RWMutex
}

// bridgeAdapter 适配器，将 *callback_bridge.Bridge 适配到 VersionUpdateSender 接口
type bridgeAdapter struct {
	bridge *callback_bridge.Bridge
}

func (a *bridgeAdapter) IsRunning() bool {
	if a.bridge == nil {
		return false
	}
	return a.bridge.IsRunning()
}

func (a *bridgeAdapter) SendVersionUpdate(payload map[string]interface{}) {
	if a.bridge != nil {
		a.bridge.SendVersionUpdate(payload)
	}
}

// NewVersionCheckService 创建版本检查服务
func NewVersionCheckService(config *VersionCheckConfig, bridge *callback_bridge.Bridge) (*VersionCheckService, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.CurrentVersion == "" {
		return nil, fmt.Errorf("current_version is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	var sender VersionUpdateSender
	if bridge != nil {
		sender = &bridgeAdapter{bridge: bridge}
	}

	svc := &VersionCheckService{
		ctx:        ctx,
		cancel:     cancel,
		checkURL:   defaultCheckURL,
		aesKey:     []byte(defaultAESKey),
		currentVer: config.CurrentVersion,
		os:         config.OS,
		arch:       config.Arch,
		language:   config.Language,
		enabled:    config.Enabled,
		bridge:     sender,
		retryCount: 0,
	}

	return svc, nil
}

// Start 启动版本检查服务
func (s *VersionCheckService) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		logging.Info("[VersionCheckService] Service disabled (AppStore version), not starting")
		return
	}

	if len(s.aesKey) == 0 {
		logging.Info("[VersionCheckService] AES key not configured, version check disabled")
		return
	}

	logging.Info("[VersionCheckService] Service started, first check in %v", initialCheckDelay)
	s.scheduleNextCheck(initialCheckDelay)
}

// Stop 停止版本检查服务
func (s *VersionCheckService) Stop() {
	s.mu.Lock()
	if s.checkTimer != nil {
		s.checkTimer.Stop()
		s.checkTimer = nil
	}
	s.mu.Unlock()

	s.cancel()
	s.wg.Wait()

	logging.Info("[VersionCheckService] Service stopped")
}

// SetLanguage 更新语言设置
func (s *VersionCheckService) SetLanguage(lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.language = lang
	logging.Info("[VersionCheckService] Language updated to: %s", lang)
}

// GetLanguage 获取当前语言设置
func (s *VersionCheckService) GetLanguage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.language
}

// scheduleNextCheck 调度下次检查
func (s *VersionCheckService) scheduleNextCheck(delay time.Duration) {
	// 调用者已持有锁时不再加锁
	if s.checkTimer != nil {
		s.checkTimer.Stop()
	}

	s.checkTimer = time.AfterFunc(delay, func() {
		s.wg.Add(1)
		defer s.wg.Done()

		// 检查 context 是否已取消
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		success := s.checkForUpdates()

		s.mu.Lock()
		defer s.mu.Unlock()

		// 再次检查 context
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if success {
			s.retryCount = 0
			logging.Info("[VersionCheckService] Next check scheduled in %v", periodicCheckDelay)
			s.scheduleNextCheck(periodicCheckDelay)
		} else {
			s.retryCount++
			// 指数退避: 1min, 2min, 4min, 8min... 最大 1 小时
			nextDelay := time.Duration(1<<(s.retryCount-1)) * time.Minute
			if nextDelay > maxRetryDelay {
				nextDelay = maxRetryDelay
			}
			logging.Info("[VersionCheckService] Check failed, retrying in %v (attempt %d)", nextDelay, s.retryCount)
			s.scheduleNextCheck(nextDelay)
		}
	})
}

// checkForUpdates 执行版本检查
func (s *VersionCheckService) checkForUpdates() bool {
	s.mu.RLock()
	os := s.os
	arch := s.arch
	lang := s.language
	currentVer := s.currentVer
	s.mu.RUnlock()

	// 构建请求 URL
	url := fmt.Sprintf("%s?os=%s&arch=%s&lang=%s", s.checkURL, os, arch, lang)
	logging.Info("[VersionCheckService] Checking updates from: %s", url)

	// 创建 HTTP 客户端
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(url)
	if err != nil {
		logging.Error("[VersionCheckService] HTTP request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.Error("[VersionCheckService] Server returned status: %d", resp.StatusCode)
		return false
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.Error("[VersionCheckService] Failed to read response: %v", err)
		return false
	}

	// 解析 JSON 响应
	var response struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		logging.Error("[VersionCheckService] Failed to parse response: %v", err)
		return false
	}

	if response.Data == "" {
		logging.Error("[VersionCheckService] No data field in response")
		return false
	}

	// 解密数据
	decrypted, err := s.decryptResponse(response.Data)
	if err != nil {
		logging.Error("[VersionCheckService] Failed to decrypt response: %v", err)
		return false
	}

	// 解析版本信息
	var versionInfo VersionInfo
	if err := json.Unmarshal(decrypted, &versionInfo); err != nil {
		logging.Error("[VersionCheckService] Failed to parse version info: %v", err)
		return false
	}

	// 检查平台支持（可选，服务端已做过滤）
	logging.Info("[VersionCheckService] Remote version: %s, current: %s", versionInfo.Version, currentVer)

	// 比较版本
	if s.compareVersions(versionInfo.Version, currentVer) > 0 {
		logging.Info("[VersionCheckService] New version available: %s", versionInfo.Version)
		s.sendVersionUpdate(&versionInfo)
	} else {
		logging.Info("[VersionCheckService] Current version is up to date")
	}

	return true
}

// decryptResponse 解密响应数据 (AES-GCM)
func (s *VersionCheckService) decryptResponse(base64Data string) ([]byte, error) {
	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("invalid data: too short")
	}

	// 提取 IV (前 12 字节)
	iv := data[:12]
	// 密文 + Tag (后续字节)
	ciphertext := data[12:]

	// 创建 AES cipher
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 解密
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// compareVersions 比较版本号 (语义化版本)
// 返回: 1 如果 v1 > v2, -1 如果 v1 < v2, 0 如果相等
func (s *VersionCheckService) compareVersions(v1, v2 string) int {
	v1Parts := parseVersion(v1)
	v2Parts := parseVersion(v2)

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(v1Parts) {
			p1 = v1Parts[i]
		}
		if i < len(v2Parts) {
			p2 = v2Parts[i]
		}

		if p1 > p2 {
			return 1
		}
		if p1 < p2 {
			return -1
		}
	}

	return 0
}

// parseVersion 解析版本号字符串为整数数组
func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		result = append(result, n)
	}
	return result
}

// sendVersionUpdate 发送版本更新回调
func (s *VersionCheckService) sendVersionUpdate(info *VersionInfo) {
	if s.bridge == nil || !s.bridge.IsRunning() {
		logging.Warning("[VersionCheckService] Bridge not available, skipping callback")
		return
	}

	payload := map[string]interface{}{
		"version":      info.Version,
		"download_url": info.DownloadURL,
		"hash":         info.Hash,
		"force_update": info.ForceUpdate,
		"change_log":   info.ChangeLog,
	}

	logging.Info("[VersionCheckService] Sending version update callback")
	s.bridge.SendVersionUpdate(payload)
}
