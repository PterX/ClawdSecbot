package scanner

import (
	"errors"
	"testing"

	"go_lib/core"
)

// mockCollector 模拟采集器，用于单元测试
type mockCollector struct {
	snapshot core.SystemSnapshot
	err      error
}

func (m *mockCollector) Collect() (core.SystemSnapshot, error) {
	return m.snapshot, m.err
}

// 确保 mockCollector 实现了 Collector 接口
var _ core.Collector = (*mockCollector)(nil)

// TestNewAssetScanner 验证扫描器创建
func TestNewAssetScanner(t *testing.T) {
	collector := &mockCollector{}
	s := NewAssetScanner(collector)
	if s == nil {
		t.Fatal("NewAssetScanner should not return nil")
	}
	if s.engine == nil {
		t.Fatal("Scanner engine should not be nil")
	}
	if s.collector == nil {
		t.Fatal("Scanner collector should not be nil")
	}
}

// TestAssetScanner_Scan_DetectsPortAndProcess 验证端口+进程规则的资产检测
func TestAssetScanner_Scan_DetectsPortAndProcess(t *testing.T) {
	collector := &mockCollector{
		snapshot: core.SystemSnapshot{
			OpenPorts: []int{22, 80, 18789},
			RunningProcesses: []core.SystemProcess{
				{Pid: 101, Name: "launchd", Cmd: "/sbin/launchd", Path: "/sbin/launchd"},
				{Pid: 502, Name: "openclaw", Cmd: "/usr/local/bin/openclaw gateway", Path: "/usr/local/bin/openclaw"},
			},
			Services:   []string{"ssh-agent"},
			FileExists: func(path string) bool { return false },
		},
	}

	s := NewAssetScanner(collector)
	s.LoadRules([]core.AssetFinderRule{
		{
			Code:      "test_port_and_process",
			Name:      "Port and Process Detection",
			LifeCycle: core.RuleLifeCycleRuntime,
			Desc:      "Detects via port and process",
			Expression: core.RuleExpression{
				Lang: "json_match",
				Expr: `{"ports": [18789], "process_keywords": ["openclaw"]}`,
			},
		},
	})

	assets, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(assets) == 0 {
		t.Fatal("Expected to detect assets, but found none")
	}

	// 验证检测到端口18789
	portFound := false
	for _, asset := range assets {
		for _, port := range asset.Ports {
			if port == 18789 {
				portFound = true
			}
		}
	}
	if !portFound {
		t.Error("Expected to detect port 18789")
	}
}

// TestAssetScanner_Scan_DetectsConfigFile 验证配置文件规则的资产检测
func TestAssetScanner_Scan_DetectsConfigFile(t *testing.T) {
	collector := &mockCollector{
		snapshot: core.SystemSnapshot{
			OpenPorts:        []int{},
			RunningProcesses: []core.SystemProcess{},
			Services:         []string{},
			FileExists: func(path string) bool {
				return path == "~/.openclaw"
			},
		},
	}

	s := NewAssetScanner(collector)
	s.LoadRules([]core.AssetFinderRule{
		{
			Code:      "test_config_file",
			Name:      "Config File Detection",
			LifeCycle: core.RuleLifeCycleStatic,
			Desc:      "Detects via config file",
			Expression: core.RuleExpression{
				Lang: "json_match",
				Expr: `{"file_paths": ["~/.openclaw", "~/.moltbot"]}`,
			},
		},
	})

	assets, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(assets) == 0 {
		t.Fatal("Expected to detect config file asset, but found none")
	}

	// 验证 metadata 中记录了配置路径
	if assets[0].Metadata["config_path"] != "~/.openclaw" {
		t.Errorf("Expected config_path '~/.openclaw', got '%s'", assets[0].Metadata["config_path"])
	}
}

// TestAssetScanner_Scan_NoMatch 验证无匹配时返回空列表
func TestAssetScanner_Scan_NoMatch(t *testing.T) {
	collector := &mockCollector{
		snapshot: core.SystemSnapshot{
			OpenPorts:        []int{22, 80},
			RunningProcesses: []core.SystemProcess{},
			Services:         []string{},
			FileExists:       func(path string) bool { return false },
		},
	}

	s := NewAssetScanner(collector)
	s.LoadRules([]core.AssetFinderRule{
		{
			Code:      "test_rule",
			Name:      "Test Rule",
			LifeCycle: core.RuleLifeCycleRuntime,
			Desc:      "Test",
			Expression: core.RuleExpression{
				Lang: "json_match",
				Expr: `{"ports": [18789]}`,
			},
		},
	})

	assets, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(assets) != 0 {
		t.Errorf("Expected 0 assets, got %d", len(assets))
	}
}

// TestAssetScanner_Scan_CollectorError 验证采集器出错时的处理
func TestAssetScanner_Scan_CollectorError(t *testing.T) {
	collector := &mockCollector{
		err: errors.New("permission denied"),
	}

	s := NewAssetScanner(collector)
	s.LoadRules([]core.AssetFinderRule{
		{
			Code:      "test_rule",
			Name:      "Test Rule",
			LifeCycle: core.RuleLifeCycleRuntime,
			Desc:      "Test",
			Expression: core.RuleExpression{
				Lang: "json_match",
				Expr: `{"ports": [18789]}`,
			},
		},
	})

	_, err := s.Scan()
	if err == nil {
		t.Fatal("Expected error from collector, got nil")
	}
}

// TestAssetScanner_Scan_MultipleRules 验证多规则同时匹配
func TestAssetScanner_Scan_MultipleRules(t *testing.T) {
	collector := &mockCollector{
		snapshot: core.SystemSnapshot{
			OpenPorts: []int{18789},
			RunningProcesses: []core.SystemProcess{
				{Pid: 502, Name: "openclaw", Cmd: "openclaw gateway", Path: "/usr/local/bin/openclaw"},
			},
			Services: []string{},
			FileExists: func(path string) bool {
				return path == "~/.openclaw"
			},
		},
	}

	s := NewAssetScanner(collector)
	s.LoadRules([]core.AssetFinderRule{
		{
			Code:      "rule_port_process",
			Name:      "Port and Process Detection",
			LifeCycle: core.RuleLifeCycleRuntime,
			Desc:      "Detects via port and process",
			Expression: core.RuleExpression{
				Lang: "json_match",
				Expr: `{"ports": [18789], "process_keywords": ["openclaw"]}`,
			},
		},
		{
			Code:      "rule_config",
			Name:      "Config File Detection",
			LifeCycle: core.RuleLifeCycleStatic,
			Desc:      "Detects via config file",
			Expression: core.RuleExpression{
				Lang: "json_match",
				Expr: `{"file_paths": ["~/.openclaw"]}`,
			},
		},
	})

	assets, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(assets) != 2 {
		t.Fatalf("Expected 2 assets from 2 rules, got %d", len(assets))
	}
}

// TestMergeAssetsByName_MultipleAssets 验证多资产合并逻辑
func TestMergeAssetsByName_MultipleAssets(t *testing.T) {
	assets := []core.Asset{
		{
			Name:         "Rule1",
			Type:         "Service",
			Ports:        []int{18789},
			ProcessPaths: []string{"/usr/local/bin/node"},
			Metadata:     map[string]string{"config_path": "/home/.openclaw"},
			ServiceName:  "openclaw-gateway",
		},
		{
			Name:         "Rule2",
			Type:         "Service",
			Ports:        []int{18789, 8080},
			ProcessPaths: []string{"/usr/local/bin/node", "/usr/local/bin/openclaw"},
			Metadata:     map[string]string{"workspace": "/home/project"},
		},
	}

	merged := MergeAssetsByName(assets, "Openclaw", "Service")
	if merged == nil {
		t.Fatal("Expected merged asset, got nil")
	}

	// 验证名称和类型
	if merged.Name != "Openclaw" {
		t.Errorf("Expected name 'Openclaw', got '%s'", merged.Name)
	}
	if merged.Type != "Service" {
		t.Errorf("Expected type 'Service', got '%s'", merged.Type)
	}

	// 验证端口合并（去重）：18789和8080
	if len(merged.Ports) != 2 {
		t.Errorf("Expected 2 unique ports, got %d: %v", len(merged.Ports), merged.Ports)
	}

	// 验证进程路径合并（去重）
	if len(merged.ProcessPaths) != 2 {
		t.Errorf("Expected 2 unique process paths, got %d: %v", len(merged.ProcessPaths), merged.ProcessPaths)
	}

	// 验证元数据合并（不覆盖）
	if merged.Metadata["config_path"] != "/home/.openclaw" {
		t.Errorf("Expected config_path '/home/.openclaw', got '%s'", merged.Metadata["config_path"])
	}
	if merged.Metadata["workspace"] != "/home/project" {
		t.Errorf("Expected workspace '/home/project', got '%s'", merged.Metadata["workspace"])
	}

	// 验证服务名称
	if merged.ServiceName != "openclaw-gateway" {
		t.Errorf("Expected service name 'openclaw-gateway', got '%s'", merged.ServiceName)
	}
}

// TestMergeAssetsByName_Empty 验证空列表合并返回nil
func TestMergeAssetsByName_Empty(t *testing.T) {
	merged := MergeAssetsByName([]core.Asset{}, "Test", "Service")
	if merged != nil {
		t.Error("Expected nil for empty assets")
	}
}

// TestMergeAssetsByName_Single 验证单个资产的合并
func TestMergeAssetsByName_Single(t *testing.T) {
	assets := []core.Asset{
		{
			Name:     "Rule1",
			Ports:    []int{18789},
			Metadata: map[string]string{"key": "value"},
		},
	}

	merged := MergeAssetsByName(assets, "Openclaw", "Service")
	if merged == nil {
		t.Fatal("Expected merged asset")
	}
	if merged.Name != "Openclaw" {
		t.Errorf("Expected name 'Openclaw', got '%s'", merged.Name)
	}
	if len(merged.Ports) != 1 || merged.Ports[0] != 18789 {
		t.Errorf("Expected port [18789], got %v", merged.Ports)
	}
	if merged.Metadata["key"] != "value" {
		t.Errorf("Expected metadata key='value', got '%s'", merged.Metadata["key"])
	}
}

// TestMergeAssetsByName_ServiceNameFromSecond 验证服务名称从第二个资产补充
func TestMergeAssetsByName_ServiceNameFromSecond(t *testing.T) {
	assets := []core.Asset{
		{
			Name:        "Rule1",
			Ports:       []int{18789},
			ServiceName: "",
			Metadata:    map[string]string{},
		},
		{
			Name:        "Rule2",
			Ports:       []int{8080},
			ServiceName: "openclaw-gateway",
			Metadata:    map[string]string{},
		},
	}

	merged := MergeAssetsByName(assets, "Openclaw", "Service")
	if merged.ServiceName != "openclaw-gateway" {
		t.Errorf("Expected service name 'openclaw-gateway', got '%s'", merged.ServiceName)
	}
}
