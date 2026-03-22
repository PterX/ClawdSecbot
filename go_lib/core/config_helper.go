package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// UpdateJSONConfig 修改 JSON 文件中的指定字段
// keyPath 支持点分路径,如 "logging.redactSensitive"
// value 是要设置的新值
func UpdateJSONConfig(filePath string, keyPath string, value interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read failed: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse failed: %v", err)
	}

	// 遍历路径并修改
	parts := strings.Split(keyPath, ".")
	current := config
	for i, part := range parts {
		if i == len(parts)-1 {
			// 最后一个节点,直接赋值
			current[part] = value
		} else {
			// 中间节点,确保存在且是 map
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				// 如果不存在或不是 map,创建新的
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		}
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %v", err)
	}

	if err := ioutil.WriteFile(filePath, newData, 0600); err != nil {
		return fmt.Errorf("write failed: %v", err)
	}

	return nil
}
