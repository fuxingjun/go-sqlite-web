package utils

import (
	"encoding/base64"
	"strings"
)

// EncodePK 将主键值（支持复合）编码为 URL 安全的 Base64 字符串
// 示例: ["1", "alice"] -> "MTp8fGFsaWNl"
func EncodePK(pk []string) string {
	raw := strings.Join(pk, ":::")
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodePK 将编码的主键解码为字符串切片
// 示例: "MTp8fGFsaWNl" -> ["1", "alice"]
func DecodePK(encoded string) ([]string, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), ":::"), nil
}
