package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/sdkerr"
	hwdns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	hwdnsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

var ErrHuaweiDNSRecordNotFound = errors.New("huawei cloud dns record not found")

type HuaweiDNSService struct {
	client *hwdns.DnsClient
}

func NewHuaweiDNSService(ak, sk, region, endpoint, projectID string) (*HuaweiDNSService, error) {
	ak = strings.TrimSpace(ak)
	sk = strings.TrimSpace(sk)
	if ak == "" || sk == "" {
		return nil, errors.New("huawei cloud dns ak and sk are required")
	}

	credBuilder := basic.NewCredentialsBuilder().
		WithAk(ak).
		WithSk(sk)
	if projectID = strings.TrimSpace(projectID); projectID != "" {
		credBuilder.WithProjectId(projectID)
	}
	credentials, err := credBuilder.SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("build huawei cloud credentials: %w", err)
	}

	httpClient, err := hwdns.DnsClientBuilder().
		WithCredential(credentials).
		WithEndpoint(normalizeHuaweiEndpoint(endpoint, region)).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("build huawei cloud dns client: %w", err)
	}
	return &HuaweiDNSService{
		client: hwdns.NewDnsClient(httpClient),
	}, nil
}

func (s *HuaweiDNSService) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	if s.client == nil {
		return nil, errors.New("huawei cloud dns client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = ctx // sdk method does not accept context in generated methods
	limit := int32(500)
	offset := int32(0)
	result := make([]ZoneInfo, 0)
	for {
		req := &hwdnsmodel.ListPublicZonesRequest{
			Type:   strPtr("public"),
			Limit:  &limit,
			Offset: &offset,
		}
		resp, err := s.client.ListPublicZones(req)
		if err != nil {
			return nil, fmt.Errorf("huawei cloud dns list zones: %w", err)
		}
		if resp == nil || resp.Zones == nil || len(*resp.Zones) == 0 {
			break
		}

		for _, zone := range *resp.Zones {
			result = append(result, ZoneInfo{
				ID:     strings.TrimSpace(ptrString(zone.Id)),
				Name:   strings.TrimSpace(ptrString(zone.Name)),
				Status: strings.TrimSpace(ptrString(zone.Status)),
			})
		}
		if len(*resp.Zones) < int(limit) {
			break
		}
		offset += int32(len(*resp.Zones))
	}

	return result, nil
}

func (s *HuaweiDNSService) CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, _ bool) (string, error) {
	if s.client == nil {
		return "", errors.New("huawei cloud dns client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = ctx
	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	name = ensureFQDN(name)
	content = strings.TrimSpace(content)
	if zoneID = strings.TrimSpace(zoneID); zoneID == "" {
		return "", errors.New("huawei cloud dns zone id is required")
	}

	if existingID, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
		return existingID, nil
	} else if !errors.Is(findErr, ErrHuaweiDNSRecordNotFound) {
		return "", fmt.Errorf("huawei cloud dns pre-check record: %w", findErr)
	}

	ttlValue := int32(300)
	_ = ttl
	req := &hwdnsmodel.CreateRecordSetRequest{
		ZoneId: zoneID,
		Body: &hwdnsmodel.CreateRecordSetRequestBody{
			Name:    name,
			Type:    recordType,
			Ttl:     &ttlValue,
			Records: []string{content},
		},
	}

	resp, err := s.client.CreateRecordSet(req)
	if err != nil {
		if existingID, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
			return existingID, nil
		}
		return "", fmt.Errorf("huawei cloud dns create record: %w", err)
	}
	if resp == nil || resp.Id == nil {
		return "", errors.New("huawei cloud dns create record returned empty record id")
	}
	return strings.TrimSpace(*resp.Id), nil
}

func (s *HuaweiDNSService) UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, _ bool) error {
	if s.client == nil {
		return errors.New("huawei cloud dns client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = ctx
	if zoneID = strings.TrimSpace(zoneID); zoneID == "" {
		return errors.New("huawei cloud dns zone id is required")
	}
	if recordID = strings.TrimSpace(recordID); recordID == "" {
		return errors.New("huawei cloud dns record id is required")
	}

	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	name = ensureFQDN(name)
	content = strings.TrimSpace(content)
	ttlValue := int32(300)
	_ = ttl
	records := []string{content}
	req := &hwdnsmodel.UpdateRecordSetsRequest{
		ZoneId:      zoneID,
		RecordsetId: recordID,
		Body: &hwdnsmodel.UpdateRecordSetsReq{
			Name:    name,
			Type:    recordType,
			Ttl:     &ttlValue,
			Records: &records,
		},
	}

	if _, err := s.client.UpdateRecordSets(req); err != nil {
		if isHuaweiNotFoundError(err) {
			return ErrHuaweiDNSRecordNotFound
		}
		return fmt.Errorf("huawei cloud dns update record: %w", err)
	}
	return nil
}

func (s *HuaweiDNSService) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if s.client == nil {
		return errors.New("huawei cloud dns client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = ctx
	if zoneID = strings.TrimSpace(zoneID); zoneID == "" {
		return errors.New("huawei cloud dns zone id is required")
	}
	if recordID = strings.TrimSpace(recordID); recordID == "" {
		return errors.New("huawei cloud dns record id is required")
	}

	req := &hwdnsmodel.DeleteRecordSetRequest{
		ZoneId:      zoneID,
		RecordsetId: recordID,
	}
	if _, err := s.client.DeleteRecordSet(req); err != nil {
		if isHuaweiNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("huawei cloud dns delete record: %w", err)
	}
	return nil
}

func (s *HuaweiDNSService) FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error) {
	if s.client == nil {
		return "", errors.New("huawei cloud dns client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = ctx
	if zoneID = strings.TrimSpace(zoneID); zoneID == "" {
		return "", errors.New("huawei cloud dns zone id is required")
	}

	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	name = ensureFQDN(name)
	content = strings.TrimSpace(content)
	limit := int32(500)
	offset := int32(0)
	searchMode := "equal"
	for {
		req := &hwdnsmodel.ListRecordSetsByZoneRequest{
			ZoneId:     zoneID,
			Limit:      &limit,
			Offset:     &offset,
			SearchMode: &searchMode,
			Type:       &recordType,
			Name:       &name,
		}
		resp, err := s.client.ListRecordSetsByZone(req)
		if err != nil {
			if isHuaweiNotFoundError(err) {
				return "", ErrHuaweiDNSRecordNotFound
			}
			return "", fmt.Errorf("huawei cloud dns find record: %w", err)
		}
		if resp == nil || resp.Recordsets == nil || len(*resp.Recordsets) == 0 {
			break
		}
		for _, recordset := range *resp.Recordsets {
			if recordset.Id == nil {
				continue
			}
			if !strings.EqualFold(normalizeFQDN(ptrString(recordset.Name)), normalizeFQDN(name)) {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(ptrString(recordset.Type)), recordType) {
				continue
			}
			if !containsStringFold(ptrStringSlice(recordset.Records), content) {
				continue
			}
			return strings.TrimSpace(*recordset.Id), nil
		}
		if len(*resp.Recordsets) < int(limit) {
			break
		}
		offset += int32(len(*resp.Recordsets))
	}
	return "", ErrHuaweiDNSRecordNotFound
}

func normalizeHuaweiEndpoint(endpoint, region string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		region = strings.TrimSpace(region)
		if region == "" {
			region = "cn-north-4"
		}
		endpoint = fmt.Sprintf("dns.%s.myhuaweicloud.com", region)
	}
	endpoint = strings.TrimSuffix(endpoint, "/")
	lower := strings.ToLower(endpoint)
	if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://") {
		endpoint = "https://" + endpoint
	}
	return endpoint
}

func isHuaweiNotFoundError(err error) bool {
	var serviceErr *sdkerr.ServiceResponseError
	if errors.As(err, &serviceErr) && serviceErr != nil {
		if serviceErr.StatusCode == 404 {
			return true
		}
		if strings.Contains(strings.ToLower(serviceErr.ErrorCode), "not") && strings.Contains(strings.ToLower(serviceErr.ErrorCode), "found") {
			return true
		}
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(raw, "not found") || strings.Contains(raw, "\"status_code\":404")
}

func ptrStringSlice(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
}
