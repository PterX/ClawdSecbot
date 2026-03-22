package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const testAESKey = "0123456789abcdef0123456789abcdef"

// TestCompareVersions 娴嬭瘯鐗堟湰鍙锋瘮杈冮€昏緫
func TestCompareVersions(t *testing.T) {
	svc := &VersionCheckService{}

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal versions", "1.0.0", "1.0.0", 0},
		{"v1 greater major", "2.0.0", "1.0.0", 1},
		{"v1 less major", "1.0.0", "2.0.0", -1},
		{"v1 greater minor", "1.2.0", "1.1.0", 1},
		{"v1 less minor", "1.1.0", "1.2.0", -1},
		{"v1 greater patch", "1.0.2", "1.0.1", 1},
		{"v1 less patch", "1.0.1", "1.0.2", -1},
		{"different length v1 longer", "1.0.0.1", "1.0.0", 1},
		{"different length v2 longer", "1.0.0", "1.0.0.1", -1},
		{"equal different length", "1.0", "1.0.0", 0},
		{"v1 greater complex", "1.2.3", "1.2.2", 1},
		{"v1 less complex", "1.2.2", "1.2.3", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%s, %s) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

// TestParseVersion 娴嬭瘯鐗堟湰鍙疯В鏋?
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{"simple version", "1.0.0", []int{1, 0, 0}},
		{"two parts", "1.0", []int{1, 0}},
		{"single part", "1", []int{1}},
		{"complex version", "1.2.3.4", []int{1, 2, 3, 4}},
		{"with invalid part", "1.a.3", []int{1, 0, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.version)
			if len(result) != len(tt.expected) {
				t.Errorf("parseVersion(%s) length = %d, want %d", tt.version, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseVersion(%s)[%d] = %d, want %d", tt.version, i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestDecryptResponse 娴嬭瘯 AES-GCM 瑙ｅ瘑
func TestDecryptResponse(t *testing.T) {
	svc := &VersionCheckService{
		aesKey: []byte(testAESKey),
	}

	// 鍒涘缓娴嬭瘯鏁版嵁
	testData := map[string]interface{}{
		"version":      "1.2.0",
		"download_url": "https://example.com/download",
		"hash":         "abc123",
		"force_update": false,
		"change_log":   "Test changelog",
	}
	plaintext, _ := json.Marshal(testData)

	// 鍔犲瘑鏁版嵁
	encrypted, err := encryptTestData(svc.aesKey, plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	// 瑙ｅ瘑骞堕獙璇?
	decrypted, err := svc.decryptResponse(encrypted)
	if err != nil {
		t.Fatalf("decryptResponse failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		t.Fatalf("Failed to unmarshal decrypted data: %v", err)
	}

	if result["version"] != testData["version"] {
		t.Errorf("version = %v, want %v", result["version"], testData["version"])
	}
	if result["download_url"] != testData["download_url"] {
		t.Errorf("download_url = %v, want %v", result["download_url"], testData["download_url"])
	}
}

// TestDecryptResponseInvalidData 娴嬭瘯鏃犳晥鏁版嵁瑙ｅ瘑
func TestDecryptResponseInvalidData(t *testing.T) {
	svc := &VersionCheckService{
		aesKey: []byte(testAESKey),
	}

	tests := []struct {
		name string
		data string
	}{
		{"invalid base64", "not-valid-base64!@#"},
		{"too short data", base64.StdEncoding.EncodeToString([]byte("short"))},
		{"invalid ciphertext", base64.StdEncoding.EncodeToString(make([]byte, 100))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.decryptResponse(tt.data)
			if err == nil {
				t.Errorf("decryptResponse(%s) should return error", tt.name)
			}
		})
	}
}

// TestNewVersionCheckService 娴嬭瘯鏈嶅姟鍒涘缓
func TestNewVersionCheckService(t *testing.T) {
	tests := []struct {
		name        string
		config      *VersionCheckConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &VersionCheckConfig{
				CurrentVersion: "1.0.0",
				OS:             "macos",
				Arch:           "arm64",
				Language:       "zh",
				Enabled:        true,
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty version",
			config: &VersionCheckConfig{
				CurrentVersion: "",
				OS:             "macos",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewVersionCheckService(tt.config, nil)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if svc == nil {
					t.Error("expected service but got nil")
				}
			}
		})
	}
}

// TestVersionCheckSetLanguage 娴嬭瘯璇█璁剧疆
func TestVersionCheckSetLanguage(t *testing.T) {
	config := &VersionCheckConfig{
		CurrentVersion: "1.0.0",
		OS:             "macos",
		Arch:           "arm64",
		Language:       "en",
		Enabled:        true,
	}

	svc, err := NewVersionCheckService(config, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// 鍒濆璇█
	if svc.GetLanguage() != "en" {
		t.Errorf("initial language = %s, want en", svc.GetLanguage())
	}

	// 鏇存柊璇█
	svc.SetLanguage("zh")
	if svc.GetLanguage() != "zh" {
		t.Errorf("updated language = %s, want zh", svc.GetLanguage())
	}
}

// TestServiceDisabled 娴嬭瘯绂佺敤鐘舵€?
func TestServiceDisabled(t *testing.T) {
	config := &VersionCheckConfig{
		CurrentVersion: "1.0.0",
		OS:             "macos",
		Arch:           "arm64",
		Language:       "en",
		Enabled:        false, // 绂佺敤
	}

	svc, err := NewVersionCheckService(config, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// 鍚姩鏈嶅姟锛堝簲璇ヤ笉浼氬紑濮嬫鏌ワ級
	svc.Start()

	// 楠岃瘉瀹氭椂鍣ㄦ湭鍚姩
	svc.mu.RLock()
	timer := svc.checkTimer
	svc.mu.RUnlock()

	if timer != nil {
		t.Error("timer should be nil when service is disabled")
	}

	svc.Stop()
}

// TestCheckForUpdatesWithMockServer 娴嬭瘯鐗堟湰妫€鏌ワ紙浣跨敤 Mock 鏈嶅姟鍣級
func TestCheckForUpdatesWithMockServer(t *testing.T) {
	// 鍑嗗娴嬭瘯鏁版嵁
	versionInfo := map[string]interface{}{
		"version":      "2.0.0",
		"download_url": "https://example.com/download",
		"hash":         "abc123",
		"force_update": false,
		"change_log":   "New version",
		"platforms": []map[string]string{
			{"os": "macos", "arch": "arm64"},
		},
	}
	plaintext, _ := json.Marshal(versionInfo)
	aesKey := []byte(testAESKey)
	encrypted, _ := encryptTestData(aesKey, plaintext)

	// 鍒涘缓 Mock 鏈嶅姟鍣?
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 楠岃瘉璇锋眰鍙傛暟
		if r.URL.Query().Get("os") != "macos" {
			t.Errorf("os param = %s, want macos", r.URL.Query().Get("os"))
		}
		if r.URL.Query().Get("arch") != "arm64" {
			t.Errorf("arch param = %s, want arm64", r.URL.Query().Get("arch"))
		}
		if r.URL.Query().Get("lang") != "zh" {
			t.Errorf("lang param = %s, want zh", r.URL.Query().Get("lang"))
		}

		response := map[string]string{"data": encrypted}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 鍒涘缓甯?Mock 鍥炶皟鐨勬湇鍔?
	var callbackReceived bool
	var receivedPayload map[string]interface{}
	var mu sync.Mutex

	mockBridge := &mockCallbackBridge{
		onSendVersionUpdate: func(payload map[string]interface{}) {
			mu.Lock()
			defer mu.Unlock()
			callbackReceived = true
			receivedPayload = payload
		},
	}

	config := &VersionCheckConfig{
		CurrentVersion: "1.0.0", // 浣庝簬杩滅▼鐗堟湰
		OS:             "macos",
		Arch:           "arm64",
		Language:       "zh",
		Enabled:        true,
	}

	svc, err := NewVersionCheckService(config, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	svc.aesKey = aesKey

	// 鏇挎崲 checkURL 鍜?bridge
	svc.checkURL = server.URL
	svc.bridge = mockBridge

	// 鎵ц妫€鏌?
	success := svc.checkForUpdates()
	if !success {
		t.Error("checkForUpdates should return true")
	}

	// 楠岃瘉鍥炶皟琚皟鐢?
	mu.Lock()
	defer mu.Unlock()
	if !callbackReceived {
		t.Error("version update callback should be called")
	}
	if receivedPayload["version"] != "2.0.0" {
		t.Errorf("callback version = %v, want 2.0.0", receivedPayload["version"])
	}
}

// TestCheckForUpdatesNoUpdate 娴嬭瘯鏃犳洿鏂版儏鍐?
func TestCheckForUpdatesNoUpdate(t *testing.T) {
	// 鍑嗗娴嬭瘯鏁版嵁锛堢増鏈浉鍚岋級
	versionInfo := map[string]interface{}{
		"version":      "1.0.0",
		"download_url": "https://example.com/download",
		"hash":         "abc123",
		"force_update": false,
		"change_log":   "Same version",
	}
	plaintext, _ := json.Marshal(versionInfo)
	aesKey := []byte(testAESKey)
	encrypted, _ := encryptTestData(aesKey, plaintext)

	// 鍒涘缓 Mock 鏈嶅姟鍣?
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"data": encrypted}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 鍒涘缓鏈嶅姟
	var callbackReceived bool
	mockBridge := &mockCallbackBridge{
		onSendVersionUpdate: func(payload map[string]interface{}) {
			callbackReceived = true
		},
	}

	config := &VersionCheckConfig{
		CurrentVersion: "1.0.0", // 涓庤繙绋嬬増鏈浉鍚?		OS:             "macos",
		Arch:           "arm64",
		Language:       "zh",
		Enabled:        true,
	}

	svc, err := NewVersionCheckService(config, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	svc.aesKey = aesKey

	svc.checkURL = server.URL
	svc.bridge = mockBridge

	// 鎵ц妫€鏌?
	success := svc.checkForUpdates()
	if !success {
		t.Error("checkForUpdates should return true")
	}

	// 楠岃瘉鍥炶皟鏈璋冪敤
	if callbackReceived {
		t.Error("version update callback should not be called when no update")
	}
}

// TestCheckForUpdatesServerError 娴嬭瘯鏈嶅姟鍣ㄩ敊璇?
func TestCheckForUpdatesServerError(t *testing.T) {
	// 鍒涘缓杩斿洖閿欒鐨?Mock 鏈嶅姟鍣?
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &VersionCheckConfig{
		CurrentVersion: "1.0.0",
		OS:             "macos",
		Arch:           "arm64",
		Language:       "zh",
		Enabled:        true,
	}

	svc, err := NewVersionCheckService(config, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	svc.aesKey = []byte(testAESKey)

	svc.checkURL = server.URL

	// 鎵ц妫€鏌ワ紙搴旇澶辫触锛?
	success := svc.checkForUpdates()
	if success {
		t.Error("checkForUpdates should return false on server error")
	}
}

// TestServiceLifecycle 娴嬭瘯鏈嶅姟鐢熷懡鍛ㄦ湡
func TestServiceLifecycle(t *testing.T) {
	config := &VersionCheckConfig{
		CurrentVersion: "1.0.0",
		OS:             "macos",
		Arch:           "arm64",
		Language:       "en",
		Enabled:        true,
	}

	svc, err := NewVersionCheckService(config, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// 鍚姩鏈嶅姟
	svc.aesKey = []byte(testAESKey)
	svc.Start()

	// 楠岃瘉瀹氭椂鍣ㄥ凡鍚姩
	svc.mu.RLock()
	timer := svc.checkTimer
	svc.mu.RUnlock()

	if timer == nil {
		t.Error("timer should not be nil after Start()")
	}

	// 鍋滄鏈嶅姟
	svc.Stop()

	// 楠岃瘉瀹氭椂鍣ㄥ凡娓呯悊
	svc.mu.RLock()
	timer = svc.checkTimer
	svc.mu.RUnlock()

	if timer != nil {
		t.Error("timer should be nil after Stop()")
	}
}

// ==================== 杈呭姪鍑芥暟鍜?Mock ====================

// encryptTestData 鍔犲瘑娴嬭瘯鏁版嵁 (AES-GCM)
func encryptTestData(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, iv, plaintext, nil)
	result := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// mockCallbackBridge 妯℃嫙鍥炶皟妗ユ帴鍣紙瀹炵幇 VersionUpdateSender 鎺ュ彛锛?
type mockCallbackBridge struct {
	running             bool
	onSendVersionUpdate func(map[string]interface{})
}

func (m *mockCallbackBridge) IsRunning() bool {
	return true
}

func (m *mockCallbackBridge) SendVersionUpdate(payload map[string]interface{}) {
	if m.onSendVersionUpdate != nil {
		m.onSendVersionUpdate(payload)
	}
}
