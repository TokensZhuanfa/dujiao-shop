package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

// randNumeric 生成指定长度的随机数字字符串。
func randNumeric(length int) string {
	if length <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			b.WriteString("0")
			continue
		}
		fmt.Fprintf(&b, "%d", n.Int64())
	}
	return b.String()
}

// randBase32 生成指定长度的大写 base32 串（A-Z + 2-7），用 crypto/rand。
// 用作流水号随机段，避免可枚举：长度 16 提供 ~80 bit 熵，远大于秒级 keyspace。
var base32Alphabet = []byte("ABCDEFGHIJKLMNPQRSTUVWXYZ23456789") // 去掉容易看错的 O / 1，留 32 字符可选

func randBase32(length int) string {
	if length <= 0 {
		return ""
	}
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(base32Alphabet))))
		if err != nil {
			out[i] = 'A'
			continue
		}
		out[i] = base32Alphabet[n.Int64()]
	}
	return string(out)
}

// generateSerialNo 生成带前缀的流水号（前缀 + 时间戳 + 16 位 base32 随机串）。
// 旧版本只有 6 位随机数字，每秒 keyspace 仅 10^6，配合时间戳可被枚举；
// 升级到 16 位 base32 后秒级 keyspace ~ 32^16 ≈ 1.2×10^24，扫描代价不可承受。
func generateSerialNo(prefix string) string {
	now := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s%s%s", prefix, now, randBase32(16))
}

// pickFirstNonEmpty 返回第一个非空（trim 后）的字符串。
func pickFirstNonEmpty(values ...string) string {
	for _, val := range values {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// appendURLQuery 向 URL 追加查询参数。
func appendURLQuery(rawURL string, params map[string]string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	for key, value := range params {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
