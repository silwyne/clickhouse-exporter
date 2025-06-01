package yaml

import (
	"os"

	"gopkg.in/yaml.v3"
)

type YamlConfig struct {
	data map[string]interface{}
}

func ReadYaml(filePath string) YamlConfig {
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		panic(err)
	}

	return YamlConfig{data: raw}
}

func (m *YamlConfig) GetObject(keys ...string) string {
	var current interface{} = m.data
	for _, key := range keys {
		if asMap, ok := current.(map[string]interface{}); ok {
			current = asMap[key]
		} else {
			return ""
		}
	}
	if str, ok := current.(string); ok {
		return str
	}
	return ""
}
