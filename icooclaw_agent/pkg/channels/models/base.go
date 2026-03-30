package models

import (
	"encoding/json"
	"strings"
)

type Allow []string

// 处理 JSON 序列化
func (a Allow) MarshalJSON() ([]byte, error) {
	return json.Marshal(a)
}

// 处理 JSON 反序列化
func (a *Allow) UnmarshalJSON(data []byte) error {
	// 传入的 JSON 字符串可能为空
	if len(data) == 0 {
		// 如果为空，直接返回 nil
		return nil
	}

	// 如果是字符串，直接赋值
	if string(data) == "[]" {
		*a = []string{}
		return nil
	}

	// 如果没有 , 则直接赋值
	if !strings.Contains(string(data), ",") {
		*a = []string{string(data)}
		return nil
	}

	// 否则，按 , 分隔
	*a = strings.Split(string(data), ",")
	return nil
}
