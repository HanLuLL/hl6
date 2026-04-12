package service

// dnsla.go implements the DNSProviderClient interface for DNSLA
// using their official HTTP API.
//
// DNSLA API docs: https://www.dnsla.com/api
// Auth: Basic Auth (API ID + API Secret)

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrDNSLARecordNotFound = errors.New("dnsla record not found")

const dnslaBaseURL = "https://api.dnsla.com/api/v1"

type DNSLAService struct {
	apiID     string
	apiSecret string
	client    *http.Client
}

func NewDNSLAService(apiID, apiSecret string) (*DNSLAService, error) {
	apiID = strings.TrimSpace(apiID)
	apiSecret = strings.TrimSpace(apiSecret)
	if apiID == "" || apiSecret == "" {
		return nil, errors.New("dnsla api_id and api_secret are required")
	}
	return &DNSLAService{
		apiID:     apiID,
		apiSecret: apiSecret,
		client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *DNSLAService) doRequest(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	url := dnslaBaseURL + path

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = strings.NewReader(string(encoded))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.apiID, s.apiSecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dnsla api request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dnsla read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrDNSLARecordNotFound
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("dnsla parse response: %w", err)
	}

	code, _ := result["code"].(float64)
	if code != 0 && resp.StatusCode >= 400 {
		msg, _ := result["message"].(string)
		if msg == "" {
			msg = fmt.Sprintf("dnsla api error code %v", code)
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrDNSLARecordNotFound
		}
		return nil, fmt.Errorf("dnsla api: %s", msg)
	}
	return result, nil
}

func (s *DNSLAService) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	result, err := s.doRequest(ctx, http.MethodGet, "/domain?page=1&pageSize=100", nil)
	if err != nil {
		return nil, fmt.Errorf("dnsla list zones: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return []ZoneInfo{}, nil
	}
	list, _ := data["list"].([]interface{})
	zones := make([]ZoneInfo, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id := extractStringOrFloat(m, "id")
		name, _ := m["domain"].(string)
		zones = append(zones, ZoneInfo{
			ID:     id,
			Name:   strings.TrimSuffix(strings.TrimSpace(name), "."),
			Status: "ENABLE",
		})
	}
	return zones, nil
}

func (s *DNSLAService) CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, _ bool) (string, error) {
	if existingID, err := s.FindRecord(ctx, zoneID, recordType, name, content); err == nil {
		return existingID, nil
	}

	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return "", err
	}
	relName, relErr := relativeRecordName(name, zoneName)
	if relErr != nil {
		return "", relErr
	}

	payload := map[string]interface{}{
		"domainId": zoneID,
		"host":     relName,
		"type":     strings.ToUpper(recordType),
		"data":     strings.TrimSpace(content),
		"ttl":      ttl,
		"lineId":   "0",
	}
	result, err := s.doRequest(ctx, http.MethodPost, "/record", payload)
	if err != nil {
		if id, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
			return id, nil
		}
		return "", fmt.Errorf("dnsla create record: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return "", errors.New("dnsla create record: empty response data")
	}
	return extractStringOrFloat(data, "id"), nil
}

func (s *DNSLAService) UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, _ bool) error {
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return err
	}
	relName, relErr := relativeRecordName(name, zoneName)
	if relErr != nil {
		return relErr
	}

	payload := map[string]interface{}{
		"id":       recordID,
		"domainId": zoneID,
		"host":     relName,
		"type":     strings.ToUpper(recordType),
		"data":     strings.TrimSpace(content),
		"ttl":      ttl,
		"lineId":   "0",
	}
	if _, err := s.doRequest(ctx, http.MethodPut, "/record/"+recordID, payload); err != nil {
		if errors.Is(err, ErrDNSLARecordNotFound) {
			return ErrDNSLARecordNotFound
		}
		return fmt.Errorf("dnsla update record: %w", err)
	}
	return nil
}

func (s *DNSLAService) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if _, err := s.doRequest(ctx, http.MethodDelete, "/record/"+recordID, nil); err != nil {
		if errors.Is(err, ErrDNSLARecordNotFound) {
			return nil
		}
		if isDNSLANotFound(err) {
			return nil
		}
		return fmt.Errorf("dnsla delete record: %w", err)
	}
	return nil
}

func (s *DNSLAService) FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error) {
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return "", err
	}
	relName, relErr := relativeRecordName(name, zoneName)
	if relErr != nil {
		return "", relErr
	}

	rtype := strings.ToUpper(strings.TrimSpace(recordType))
	content = strings.TrimSpace(content)

	result, err := s.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/record?domainId=%s&host=%s&type=%s&page=1&pageSize=500", zoneID, relName, rtype), nil)
	if err != nil {
		if errors.Is(err, ErrDNSLARecordNotFound) || isDNSLANotFound(err) {
			return "", ErrDNSLARecordNotFound
		}
		return "", fmt.Errorf("dnsla find record: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return "", ErrDNSLARecordNotFound
	}
	list, _ := data["list"].([]interface{})
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		itemType, _ := m["type"].(string)
		if !strings.EqualFold(strings.TrimSpace(itemType), rtype) {
			continue
		}
		itemData, _ := m["data"].(string)
		if strings.EqualFold(strings.TrimSpace(itemData), content) {
			return extractStringOrFloat(m, "id"), nil
		}
	}
	return "", ErrDNSLARecordNotFound
}

func (s *DNSLAService) zoneNameByID(ctx context.Context, zoneID string) (string, error) {
	zones, err := s.ListZones(ctx)
	if err != nil {
		return "", err
	}
	for _, z := range zones {
		if z.ID == zoneID {
			return z.Name, nil
		}
	}
	return "", fmt.Errorf("dnsla zone %q not found", zoneID)
}

func isDNSLANotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404") || strings.Contains(msg, "no record")
}

func extractStringOrFloat(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return strconv.FormatInt(int64(v), 10)
	}
	return ""
}
