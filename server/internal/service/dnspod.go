package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tprofile "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

var ErrDNSPodRecordNotFound = errors.New("dnspod record not found")

type DNSPodService struct {
	client *dnspod.Client
}

func NewDNSPodService(secretID, secretKey, region string) (*DNSPodService, error) {
	secretID = strings.TrimSpace(secretID)
	secretKey = strings.TrimSpace(secretKey)
	if secretID == "" || secretKey == "" {
		return nil, errors.New("dnspod secret_id and secret_key are required")
	}

	httpProfile := tprofile.NewHttpProfile()
	httpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	clientProfile := tprofile.NewClientProfile()
	clientProfile.HttpProfile = httpProfile

	client, err := dnspod.NewClient(tcommon.NewCredential(secretID, secretKey), strings.TrimSpace(region), clientProfile)
	if err != nil {
		return nil, fmt.Errorf("create dnspod client: %w", err)
	}
	return &DNSPodService{client: client}, nil
}

func (s *DNSPodService) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	if s.client == nil {
		return nil, errors.New("dnspod client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	limit := int64(3000)
	offset := int64(0)
	result := make([]ZoneInfo, 0)
	for {
		req := dnspod.NewDescribeDomainListRequest()
		req.Type = strPtr("ALL")
		req.Limit = &limit
		req.Offset = &offset

		resp, err := s.client.DescribeDomainListWithContext(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("dnspod list zones: %w", err)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.DomainList) == 0 {
			break
		}

		for _, item := range resp.Response.DomainList {
			if item == nil || item.DomainId == nil || item.Name == nil {
				continue
			}
			result = append(result, ZoneInfo{
				ID:     strconv.FormatUint(*item.DomainId, 10),
				Name:   strings.TrimSpace(*item.Name),
				Status: strings.TrimSpace(ptrString(item.Status)),
			})
		}

		if len(resp.Response.DomainList) < int(limit) {
			break
		}
		offset += int64(len(resp.Response.DomainList))
	}
	return result, nil
}

func (s *DNSPodService) CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, _ bool) (string, error) {
	if s.client == nil {
		return "", errors.New("dnspod client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	domainID, err := parseDNSPodDomainID(zoneID)
	if err != nil {
		return "", err
	}
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return "", err
	}
	subDomain, err := relativeRecordName(name, zoneName)
	if err != nil {
		return "", err
	}

	if existingID, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
		return existingID, nil
	} else if !errors.Is(findErr, ErrDNSPodRecordNotFound) {
		return "", fmt.Errorf("dnspod pre-check record: %w", findErr)
	}

	req := dnspod.NewCreateRecordRequest()
	req.Domain = strPtr(zoneName)
	req.DomainId = &domainID
	req.RecordType = strPtr(strings.ToUpper(strings.TrimSpace(recordType)))
	req.RecordLine = strPtr("默认")
	req.Value = strPtr(strings.TrimSpace(content))
	req.SubDomain = strPtr(subDomain)
	_ = ttl

	resp, err := s.client.CreateRecordWithContext(ctx, req)
	if err != nil {
		if existingID, findErr := s.FindRecord(ctx, zoneID, recordType, name, content); findErr == nil {
			return existingID, nil
		}
		return "", fmt.Errorf("dnspod create record: %w", err)
	}
	if resp == nil || resp.Response == nil || resp.Response.RecordId == nil {
		return "", errors.New("dnspod create record returned empty record id")
	}
	return strconv.FormatUint(*resp.Response.RecordId, 10), nil
}

func (s *DNSPodService) UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, _ bool) error {
	if s.client == nil {
		return errors.New("dnspod client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	domainID, err := parseDNSPodDomainID(zoneID)
	if err != nil {
		return err
	}
	parsedRecordID, err := strconv.ParseUint(strings.TrimSpace(recordID), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid dnspod record id %q", recordID)
	}
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return err
	}
	subDomain, err := relativeRecordName(name, zoneName)
	if err != nil {
		return err
	}

	req := dnspod.NewModifyRecordRequest()
	req.Domain = strPtr(zoneName)
	req.DomainId = &domainID
	req.RecordId = &parsedRecordID
	req.RecordType = strPtr(strings.ToUpper(strings.TrimSpace(recordType)))
	req.RecordLine = strPtr("默认")
	req.Value = strPtr(strings.TrimSpace(content))
	req.SubDomain = strPtr(subDomain)
	_ = ttl

	if _, err := s.client.ModifyRecordWithContext(ctx, req); err != nil {
		if isDNSPodNotFoundError(err) {
			return ErrDNSPodRecordNotFound
		}
		return fmt.Errorf("dnspod update record: %w", err)
	}
	return nil
}

func (s *DNSPodService) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if s.client == nil {
		return errors.New("dnspod client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	domainID, err := parseDNSPodDomainID(zoneID)
	if err != nil {
		return err
	}
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return err
	}
	parsedRecordID, err := strconv.ParseUint(strings.TrimSpace(recordID), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid dnspod record id %q", recordID)
	}

	req := dnspod.NewDeleteRecordRequest()
	req.Domain = strPtr(zoneName)
	req.DomainId = &domainID
	req.RecordId = &parsedRecordID
	if _, err := s.client.DeleteRecordWithContext(ctx, req); err != nil {
		if isDNSPodNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("dnspod delete record: %w", err)
	}
	return nil
}

func (s *DNSPodService) FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error) {
	if s.client == nil {
		return "", errors.New("dnspod client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	domainID, err := parseDNSPodDomainID(zoneID)
	if err != nil {
		return "", err
	}
	zoneName, err := s.zoneNameByID(ctx, zoneID)
	if err != nil {
		return "", err
	}
	subDomain, err := relativeRecordName(name, zoneName)
	if err != nil {
		return "", err
	}

	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	content = strings.TrimSpace(content)
	offset := uint64(0)
	limit := uint64(3000)
	for {
		req := dnspod.NewDescribeRecordListRequest()
		req.Domain = &zoneName
		req.DomainId = &domainID
		req.Subdomain = &subDomain
		req.RecordType = &recordType
		req.ErrorOnEmpty = strPtr("no")
		req.Offset = &offset
		req.Limit = &limit

		resp, err := s.client.DescribeRecordListWithContext(ctx, req)
		if err != nil {
			if isDNSPodNotFoundError(err) {
				return "", ErrDNSPodRecordNotFound
			}
			return "", fmt.Errorf("dnspod find record: %w", err)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.RecordList) == 0 {
			break
		}

		for _, item := range resp.Response.RecordList {
			if item == nil || item.RecordId == nil {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(ptrString(item.Type)), recordType) {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(ptrString(item.Value)), content) {
				continue
			}
			return strconv.FormatUint(*item.RecordId, 10), nil
		}

		if len(resp.Response.RecordList) < int(limit) {
			break
		}
		offset += uint64(len(resp.Response.RecordList))
	}
	return "", ErrDNSPodRecordNotFound
}

func (s *DNSPodService) zoneNameByID(ctx context.Context, zoneID string) (string, error) {
	domainID, err := parseDNSPodDomainID(zoneID)
	if err != nil {
		return "", err
	}

	req := dnspod.NewDescribeDomainRequest()
	req.DomainId = &domainID
	resp, err := s.client.DescribeDomainWithContext(ctx, req)
	if err == nil && resp != nil && resp.Response != nil && resp.Response.DomainInfo != nil && resp.Response.DomainInfo.Domain != nil {
		return strings.TrimSpace(*resp.Response.DomainInfo.Domain), nil
	}
	if err != nil && !isDNSPodMissingDomainParameterError(err) {
		return "", fmt.Errorf("dnspod describe domain: %w", err)
	}

	name, fallbackErr := s.zoneNameByIDFromList(ctx, domainID)
	if fallbackErr != nil {
		if err != nil {
			return "", fmt.Errorf("dnspod describe domain: %w; fallback list failed: %v", err, fallbackErr)
		}
		return "", fallbackErr
	}
	return name, nil
}

func (s *DNSPodService) zoneNameByIDFromList(ctx context.Context, domainID uint64) (string, error) {
	limit := int64(3000)
	offset := int64(0)
	for {
		req := dnspod.NewDescribeDomainListRequest()
		req.Type = strPtr("ALL")
		req.Limit = &limit
		req.Offset = &offset

		resp, err := s.client.DescribeDomainListWithContext(ctx, req)
		if err != nil {
			return "", fmt.Errorf("dnspod list domain fallback: %w", err)
		}
		if resp == nil || resp.Response == nil || len(resp.Response.DomainList) == 0 {
			break
		}
		for _, item := range resp.Response.DomainList {
			if item == nil || item.DomainId == nil || item.Name == nil {
				continue
			}
			if *item.DomainId == domainID {
				return strings.TrimSpace(*item.Name), nil
			}
		}

		if len(resp.Response.DomainList) < int(limit) {
			break
		}
		offset += int64(len(resp.Response.DomainList))
	}
	return "", fmt.Errorf("dnspod zone %d not found", domainID)
}

func parseDNSPodDomainID(zoneID string) (uint64, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(zoneID), 10, 64)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid dnspod zone id %q", zoneID)
	}
	return parsed, nil
}

func isDNSPodNotFoundError(err error) bool {
	var sdkErr *tcerr.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		code := strings.ToLower(strings.TrimSpace(sdkErr.GetCode()))
		if strings.Contains(code, "recordidinvalid") || strings.Contains(code, "recordnot") || strings.Contains(code, "notexists") {
			return true
		}
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "record not found") || strings.Contains(msg, "recordidinvalid")
}

func isDNSPodMissingDomainParameterError(err error) bool {
	if err == nil {
		return false
	}
	var sdkErr *tcerr.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		code := strings.ToLower(strings.TrimSpace(sdkErr.GetCode()))
		msg := strings.ToLower(strings.TrimSpace(sdkErr.GetMessage()))
		if code == "missingparameter" && strings.Contains(msg, "domain") {
			return true
		}
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(raw, "missingparameter") && strings.Contains(raw, "domain")
}

func strPtr(v string) *string {
	return &v
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
