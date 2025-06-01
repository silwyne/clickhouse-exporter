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

func (m *YamlConfig) GetValues(keys ...string) YamlConfig {
	var current interface{} = m.data
	for _, key := range keys {
		if asMap, ok := current.(map[string]interface{}); ok {
			current = asMap[key]
		} else {
			return YamlConfig{nil}
		}
	}
	var parsed_value = current.(map[string]interface{})
	return YamlConfig{data: parsed_value}
}

func (m *YamlConfig) Contains(key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *YamlConfig) GetData() map[string]interface{} {
	return m.data
}
