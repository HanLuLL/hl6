package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// StringSlice 存 PostgreSQL jsonb 字符串数组。
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("StringSlice: unsupported type %T", value)
	}
	var out []string
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

// UintSlice 存 PostgreSQL jsonb 无符号整数数组。
type UintSlice []uint

func (s UintSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *UintSlice) Scan(value interface{}) error {
	if value == nil {
		*s = UintSlice{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("UintSlice: unsupported type %T", value)
	}
	var out []uint
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

// MatchedRuleHit 单条规则命中证据（写入 subdomain_scans.matched_rules）。
type MatchedRuleHit struct {
	RuleID   uint   `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Action   string `json:"action"`
	Snippet  string `json:"snippet"`
}

// MatchedRulesSlice jsonb 数组包装。
type MatchedRulesSlice []MatchedRuleHit

func (s MatchedRulesSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *MatchedRulesSlice) Scan(value interface{}) error {
	if value == nil {
		*s = MatchedRulesSlice{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("MatchedRulesSlice: unsupported type")
	}
	var out []MatchedRuleHit
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

// FetchChannelDetail 单协议抓取摘要（写入 subdomain_scans.fetch_details）。
type FetchChannelDetail struct {
	Scheme         string `json:"scheme"`
	RequestURL     string `json:"request_url"`
	Status         string `json:"status"`
	HTTPStatusCode int    `json:"http_status_code"`
	FinalURL       string `json:"final_url"`
	ErrorMessage   string `json:"error_message,omitempty"`
	TitlePreview   string `json:"title_preview,omitempty"`
}

// DualFetchDetails HTTP/HTTPS 双通道抓取摘要。
type DualFetchDetails struct {
	HTTPS FetchChannelDetail `json:"https"`
	HTTP  FetchChannelDetail `json:"http"`
}

// DualFetchDetailsJSON 可空 jsonb 列包装。
type DualFetchDetailsJSON DualFetchDetails

func (d DualFetchDetailsJSON) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	return string(b), err
}

func (d *DualFetchDetailsJSON) Scan(value interface{}) error {
	if value == nil {
		*d = DualFetchDetailsJSON{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("DualFetchDetailsJSON: unsupported type %T", value)
	}
	var out DualFetchDetails
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*d = DualFetchDetailsJSON(out)
	return nil
}
