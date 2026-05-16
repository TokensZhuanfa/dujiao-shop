package shared

import "time"

// ParseTimeNullable 解析可空的 RFC3339 时间字符串，空串返回 nil。
func ParseTimeNullable(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
