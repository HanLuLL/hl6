package service

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
)

type CloudflareService struct {
	client *cloudflare.Client
}

func NewCloudflareService(apiToken string) *CloudflareService {
	if apiToken == "" {
		return &CloudflareService{client: nil}
	}
	client := cloudflare.NewClient(option.WithAPIToken(apiToken))
	return &CloudflareService{client: client}
}

func (s *CloudflareService) buildNewBody(recordType, name, content string, ttl int, proxied bool) dns.RecordNewParamsBodyUnion {
	switch recordType {
	case "AAAA":
		return dns.AAAARecordParam{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.AAAARecordTypeAAAA),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxied),
		}
	case "CNAME":
		return dns.CNAMERecordParam{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.CNAMERecordTypeCNAME),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxied),
		}
	default: // A
		return dns.ARecordParam{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.ARecordTypeA),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxied),
		}
	}
}

func (s *CloudflareService) buildUpdateBody(recordType, name, content string, ttl int, proxied bool) dns.RecordUpdateParamsBodyUnion {
	switch recordType {
	case "AAAA":
		return dns.AAAARecordParam{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.AAAARecordTypeAAAA),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxied),
		}
	case "CNAME":
		return dns.CNAMERecordParam{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.CNAMERecordTypeCNAME),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxied),
		}
	default: // A
		return dns.ARecordParam{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.ARecordTypeA),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxied),
		}
	}
}

func (s *CloudflareService) CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, proxied bool) (string, error) {
	if s.client == nil {
		return "mock-record-id", nil
	}

	record, err := s.client.DNS.Records.New(ctx, dns.RecordNewParams{
		ZoneID: cloudflare.F(zoneID),
		Body:   s.buildNewBody(recordType, name, content, ttl, proxied),
	})
	if err != nil {
		return "", fmt.Errorf("cloudflare create record: %w", err)
	}
	return record.ID, nil
}

func (s *CloudflareService) UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, proxied bool) error {
	if s.client == nil {
		return nil
	}

	_, err := s.client.DNS.Records.Update(ctx, recordID, dns.RecordUpdateParams{
		ZoneID: cloudflare.F(zoneID),
		Body:   s.buildUpdateBody(recordType, name, content, ttl, proxied),
	})
	if err != nil {
		return fmt.Errorf("cloudflare update record: %w", err)
	}
	return nil
}

func (s *CloudflareService) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	if s.client == nil {
		return nil
	}

	_, err := s.client.DNS.Records.Delete(ctx, recordID, dns.RecordDeleteParams{
		ZoneID: cloudflare.F(zoneID),
	})
	if err != nil {
		return fmt.Errorf("cloudflare delete record: %w", err)
	}
	return nil
}
