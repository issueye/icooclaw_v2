package envmgr

import (
	"encoding/json"
	"sort"
)

type Manager struct {
	values map[string]string
}

func New(initial map[string]string) *Manager {
	m := &Manager{
		values: make(map[string]string),
	}
	for key, value := range initial {
		if key == "" {
			continue
		}
		m.values[key] = value
	}
	return m
}

func (m *Manager) Merge(extra map[string]string) *Manager {
	if m == nil {
		return New(extra)
	}

	merged := New(m.values)
	for key, value := range extra {
		if key == "" {
			continue
		}
		merged.values[key] = value
	}
	return merged
}

func (m *Manager) ToMap() map[string]string {
	if m == nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(m.values))
	for key, value := range m.values {
		result[key] = value
	}
	return result
}

func (m *Manager) ToList() []string {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m.values))
	for key := range m.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+m.values[key])
	}
	return result
}

func ParseJSON(content string) (map[string]string, error) {
	if content == "" {
		return map[string]string{}, nil
	}

	result := map[string]string{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func MustJSON(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(data)
}
