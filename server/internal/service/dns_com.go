package service

// dns_com.go implements the DNSProviderClient interface for DNS.com
// using their official HTTP API.
//
// DNS.com API docs: https://www.dns.com/supports/api/
// Auth: HMAC-SHA256 signature (apiId + apiKey + timestamp + nonce)

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrDNSComRecordNotFound = errors.New("dns.com record not found")

const dnsComBaseURL = "https://www.dns.com/api"

type DNSComService struct {
	apiID  string
	apiKey string
	client *http.Client
}

func NewDNSComService(apiID, apiKey string) (*DNSComService, error) {
	apiID = strings.TrimSpace(apiID)
	apiKey = strings.TrimSpace(apiKey)
	if apiID == "" || apiKey == "" {
		return nil, errors.New("dns_com api_id and api_key are required")
	}
	return &DNSComService{
		apiID:  apiID,
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *DNSComService) sign(timestamp, nonce string) string {
	data := s.apiID + s.apiKey + timestamp + nonce
	mac := hmac.New(sha256.New, []byte(s.apiKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *DNSComService) commonParams() map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := strconv.Itoa(rand.Intn(1000000))
	return map[string]string{
		"apiId":     s.apiID,
		"timestamp": timestamp,
		"nonce":     nonce,
		"signature": s.sign(timestamp, nonce),
	}
}

func (s *DNSComService) doRequest(ctx context.Context, method, path string, params map[string]string) (map[string]interface{}, error) {
	url := dnsComBaseURL + path

	body := make(map[string]string)
	for k, v := range s.commonParams() {
		body[k] = v
	}
	for k, v := range params {
		body[k] = v
	}

	encoded, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(encoded)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dns.com api request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dns.com read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("dns.com parse response: %w", err)
	}

	code, _ := result["code"].(float64)
	if code != 0 {
		msg, _ := result["message"].(string)
		if msg == "" {
			msg = fmt.Sprintf("dns.com api error code %v", code)
		}
		return nil, fmt.Errorf("dns.com api: %s", msg)
	}
	return result, nil
}

func (s *DNSComService) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	result, err := s.doRequest(ctx, http.MethodPost, "/domain/list/", map[string]string{
		"page":     "1",
		"pagesize": "100",
	})
	if err != nil {
		return nil, fmt.Errorf("dns.com list zones: %w", err)
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
		id, _ := m["domain_id"].(string)
		if id == "" {
			if idNum, ok := m["domain_id"].(float64); ok {
				id = strconv.FormatInt(int64(idNum), 10)
			}
		}
		name, _ := m["domain"].(string)
		status := "ENABLE"
		zones = append(zones, ZoneInfo{ID: id, Name: strings.TrimSuffix(name, "."), Status: status})
	}
	return zones, nil
}

func (s *DNSComService) CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, _ bool) (string, error) {
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

	result, err := s.doRequest(ctx, http.MethodPost, "/record/create/", map[string]string{
		"domain_id": zoneID,
		"host":      relName,
		"type":      strings.ToUpper(recordType),
		"data":      strings.TrimSpace(content),
		"ttl":       strconv.Itoa(ttl),
		"line_id":   "0",
	})
	if err != nil {
		if id, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
			return id, nil
		}
		return "", fmt.Errorf("dns.com create record: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return "", errors.New("dns.com create record: empty response data")
	}
	id, _ := data["record_id"].(string)
	if id == "" {
		if idNum, ok := data["record_id"].(float64); ok {
			id = strconv.FormatInt(int64(idNum), 10)
		}
	}
	return id, nil
}

func (s *DNSComService) UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, _ bool) error {
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return err
	}
	relName, relErr := relativeRecordName(name, zoneName)
	if relErr != nil {
		return relErr
	}

	if _, err := s.doRequest(ctx, http.MethodPost, "/record/modify/", map[string]string{
		"domain_id": zoneID,
		"record_id": recordID,
		"host":      relName,
		"type":      strings.ToUpper(recordType),
		"data":      strings.TrimSpace(content),
		"ttl":       strconv.Itoa(ttl),
		"line_id":   "0",
	}); err != nil {
		if isDNSComNotFound(err) {
			return ErrDNSComRecordNotFound
		}
		return fmt.Errorf("dns.com update record: %w", err)
	}
	return nil
}

func (s *DNSComService) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if _, err := s.doRequest(ctx, http.MethodPost, "/record/remove/", map[string]string{
		"domain_id": zoneID,
		"record_id": recordID,
	}); err != nil {
		if isDNSComNotFound(err) {
			return nil
		}
		return fmt.Errorf("dns.com delete record: %w", err)
	}
	return nil
}

func (s *DNSComService) FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error) {
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

	result, err := s.doRequest(ctx, http.MethodPost, "/record/list/", map[string]string{
		"domain_id": zoneID,
		"host":      relName,
		"page":      "1",
		"pagesize":  "500",
	})
	if err != nil {
		return "", fmt.Errorf("dns.com find record: %w", err)
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return "", ErrDNSComRecordNotFound
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
			id, _ := m["record_id"].(string)
			if id == "" {
				if idNum, ok := m["record_id"].(float64); ok {
					id = strconv.FormatInt(int64(idNum), 10)
				}
			}
			return id, nil
		}
	}
	return "", ErrDNSComRecordNotFound
}

func (s *DNSComService) zoneNameByID(ctx context.Context, zoneID string) (string, error) {
	zones, err := s.ListZones(ctx)
	if err != nil {
		return "", err
	}
	for _, z := range zones {
		if z.ID == zoneID {
			return z.Name, nil
		}
	}
	return "", fmt.Errorf("dns.com zone %q not found", zoneID)
}

func isDNSComNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no record") || strings.Contains(msg, "404")
}
