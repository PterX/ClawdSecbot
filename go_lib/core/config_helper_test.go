package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateJSONConfigUpdatesExistingNestedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"logging":{"redactSensitive":false}}`), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	if err := UpdateJSONConfig(path, "logging.redactSensitive", true); err != nil {
		t.Fatalf("UpdateJSONConfig failed: %v", err)
	}

	config := readConfigMap(t, path)
	logging, ok := config["logging"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected logging object, got %#v", config["logging"])
	}
	if logging["redactSensitive"] != true {
		t.Fatalf("Expected redactSensitive=true, got %#v", logging["redactSensitive"])
	}
}

func TestUpdateJSONConfigCreatesMissingNestedPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"name":"bot"}`), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	if err := UpdateJSONConfig(path, "security.audit.enabled", true); err != nil {
		t.Fatalf("UpdateJSONConfig failed: %v", err)
	}

	config := readConfigMap(t, path)
	security, ok := config["security"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected security object, got %#v", config["security"])
	}
	audit, ok := security["audit"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected audit object, got %#v", security["audit"])
	}
	if audit["enabled"] != true {
		t.Fatalf("Expected audit.enabled=true, got %#v", audit["enabled"])
	}
	if config["name"] != "bot" {
		t.Fatalf("Expected unrelated fields to stay intact, got %#v", config["name"])
	}
}

func TestUpdateJSONConfigReturnsParseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{invalid`), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	if err := UpdateJSONConfig(path, "security.enabled", true); err == nil {
		t.Fatal("Expected parse error")
	}
}

func readConfigMap(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}
	return config
}
