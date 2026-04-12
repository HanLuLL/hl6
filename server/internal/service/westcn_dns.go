package service

// westcn_dns.go implements the DNSProviderClient interface for 西部数码 (WestCN)
// using their official HTTP API.
//
// WestCN API docs: https://www.west.cn/docs/
// Auth: API username + password (Base64 Basic Auth or token-based)

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrWestCNRecordNotFound = errors.New("westcn dns record not found")

const westcnBaseURL = "https://api.west.cn/API/v2/domain"

type WestCNDNSService struct {
	username string
	password string
	client   *http.Client
}

func NewWestCNDNSService(username, password string) (*WestCNDNSService, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil, errors.New("westcn_dns username and password are required")
	}
	return &WestCNDNSService{
		username: username,
		password: password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *WestCNDNSService) baseParams() url.Values {
	params := url.Values{}
	params.Set("username", s.username)
	params.Set("password", s.password)
	return params
}

func (s *WestCNDNSService) doRequest(ctx context.Context, act string, extraParams map[string]string) (map[string]interface{}, error) {
	params := s.baseParams()
	params.Set("act", act)
	for k, v := range extraParams {
		params.Set(k, v)
	}

	reqURL := westcnBaseURL + "/dns/?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("westcn dns api request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("westcn dns read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("westcn dns parse response: %w", err)
	}

	code, _ := result["result"].(float64)
	if code != 200 {
		msg, _ := result["msg"].(string)
		if msg == "" {
			msg = fmt.Sprintf("westcn dns api error code %v", code)
		}
		if code == 404 {
			return nil, ErrWestCNRecordNotFound
		}
		return nil, fmt.Errorf("westcn dns api: %s", msg)
	}
	return result, nil
}

func (s *WestCNDNSService) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	result, err := s.doRequest(ctx, "getdomainlist", map[string]string{
		"page":     "1",
		"pagesize": "100",
	})
	if err != nil {
		return nil, fmt.Errorf("westcn dns list zones: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return []ZoneInfo{}, nil
	}
	items, _ := data["items"].([]interface{})
	zones := make([]ZoneInfo, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["domain"].(string)
		zones = append(zones, ZoneInfo{
			ID:     strings.TrimSuffix(strings.TrimSpace(name), "."),
			Name:   strings.TrimSuffix(strings.TrimSpace(name), "."),
			Status: "ENABLE",
		})
	}
	return zones, nil
}

func (s *WestCNDNSService) CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, _ bool) (string, error) {
	if existingID, err := s.FindRecord(ctx, zoneID, recordType, name, content); err == nil {
		return existingID, nil
	}

	relName, relErr := relativeRecordName(name, zoneID)
	if relErr != nil {
		return "", relErr
	}

	result, err := s.doRequest(ctx, "dnsadd", map[string]string{
		"domain":  zoneID,
		"host":    relName,
		"type":    strings.ToUpper(recordType),
		"value":   strings.TrimSpace(content),
		"ttl":     strconv.Itoa(ttl),
		"line":    "0",
	})
	if err != nil {
		if id, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
			return id, nil
		}
		return "", fmt.Errorf("westcn dns create record: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return "", errors.New("westcn dns create record: empty response data")
	}
	id := extractStringOrFloat(data, "id")
	if id == "" {
		id = extractStringOrFloat(data, "record_id")
	}
	return id, nil
}

func (s *WestCNDNSService) UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, _ bool) error {
	relName, relErr := relativeRecordName(name, zoneID)
	if relErr != nil {
		return relErr
	}

	if _, err := s.doRequest(ctx, "dnsmodify", map[string]string{
		"domain":    zoneID,
		"id":        recordID,
		"host":      relName,
		"type":      strings.ToUpper(recordType),
		"value":     strings.TrimSpace(content),
		"ttl":       strconv.Itoa(ttl),
		"line":      "0",
	}); err != nil {
		if errors.Is(err, ErrWestCNRecordNotFound) {
			return ErrWestCNRecordNotFound
		}
		return fmt.Errorf("westcn dns update record: %w", err)
	}
	return nil
}

func (s *WestCNDNSService) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if _, err := s.doRequest(ctx, "dnsdel", map[string]string{
		"domain": zoneID,
		"id":     recordID,
	}); err != nil {
		if errors.Is(err, ErrWestCNRecordNotFound) || isWestCNNotFound(err) {
			return nil
		}
		return fmt.Errorf("westcn dns delete record: %w", err)
	}
	return nil
}

func (s *WestCNDNSService) FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error) {
	relName, relErr := relativeRecordName(name, zoneID)
	if relErr != nil {
		return "", relErr
	}

	rtype := strings.ToUpper(strings.TrimSpace(recordType))
	content = strings.TrimSpace(content)

	result, err := s.doRequest(ctx, "dnslist", map[string]string{
		"domain": zoneID,
		"host":   relName,
	})
	if err != nil {
		if errors.Is(err, ErrWestCNRecordNotFound) || isWestCNNotFound(err) {
			return "", ErrWestCNRecordNotFound
		}
		return "", fmt.Errorf("westcn dns find record: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return "", ErrWestCNRecordNotFound
	}
	items, _ := data["items"].([]interface{})
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		itemType, _ := m["type"].(string)
		if !strings.EqualFold(strings.TrimSpace(itemType), rtype) {
			continue
		}
		itemValue, _ := m["value"].(string)
		if strings.EqualFold(strings.TrimSpace(itemValue), content) {
			id := extractStringOrFloat(m, "id")
			if id == "" {
				id = extractStringOrFloat(m, "record_id")
			}
			return id, nil
		}
	}
	return "", ErrWestCNRecordNotFound
}

func isWestCNNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404") || strings.Contains(msg, "no record")
}
