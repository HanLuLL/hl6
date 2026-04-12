package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"hl6-server/internal/model"
)

var ErrProviderNotImplemented = errors.New("provider not implemented in this build")

type DNSProviderClient interface {
	ListZones(ctx context.Context) ([]ZoneInfo, error)
	CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, proxied bool) (string, error)
	UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, proxied bool) error
	DeleteRecord(ctx context.Context, zoneID, recordID string) error
	FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error)
}

func ParseProviderCredentials(provider, raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("provider credentials are empty")
	}

	// Cloudflare supports legacy plain token for backward compatibility.
	if model.NormalizeProvider(provider) == model.DNSProviderCloudflare && !strings.HasPrefix(trimmed, "{") {
		return map[string]string{"api_token": trimmed}, nil
	}

	credentials := map[string]string{}
	if err := json.Unmarshal([]byte(trimmed), &credentials); err != nil {
		return nil, errors.New("provider credentials must be a json object")
	}
	for k, v := range credentials {
		credentials[k] = strings.TrimSpace(v)
	}
	return credentials, nil
}


func BuildProviderClient(provider string, credentials map[string]string) (DNSProviderClient, error) {
	switch model.NormalizeProvider(provider) {
	case model.DNSProviderCloudflare:
		token := pickCredential(credentials, "api_token")
		if token == "" {
			return nil, errors.New("cloudflare api_token is required")
		}
		return NewCloudflareService(token)

	case model.DNSProviderDNSPod:
		secretID := pickCredential(credentials, "secret_id", "secretid", "access_key_id", "ak")
		secretKey := pickCredential(credentials, "secret_key", "secretkey", "access_key_secret", "sk")
		region := pickCredential(credentials, "region", "region_id")
		if secretID == "" || secretKey == "" {
			return nil, errors.New("dnspod secret_id and secret_key are required")
		}
		return NewDNSPodService(secretID, secretKey, region)

	case model.DNSProviderAliDNS:
		accessKeyID := pickCredential(credentials, "access_key_id", "ak", "secret_id")
		accessKeySecret := pickCredential(credentials, "access_key_secret", "sk", "secret_key")
		regionID := pickCredential(credentials, "region_id", "region")
		endpoint := pickCredential(credentials, "endpoint")
		if accessKeyID == "" || accessKeySecret == "" {
			return nil, errors.New("aliyun dns access_key_id and access_key_secret are required")
		}
		return NewAliDNSService(accessKeyID, accessKeySecret, regionID, endpoint)

	case model.DNSProviderHuaweiDNS:
		ak := pickCredential(credentials, "ak", "access_key_id", "secret_id")
		sk := pickCredential(credentials, "sk", "access_key_secret", "secret_key")
		region := pickCredential(credentials, "region", "region_id")
		endpoint := pickCredential(credentials, "endpoint")
		projectID := pickCredential(credentials, "project_id")
		if ak == "" || sk == "" {
			return nil, errors.New("huawei cloud dns ak and sk are required")
		}
		return NewHuaweiDNSService(ak, sk, region, endpoint, projectID)

	case model.DNSProviderAWSRoute53:
		accessKeyID := pickCredential(credentials, "access_key_id", "ak")
		accessKeySecret := pickCredential(credentials, "access_key_secret", "secret_key", "sk")
		region := pickCredential(credentials, "region", "region_id")
		if accessKeyID == "" || accessKeySecret == "" {
			return nil, errors.New("aws_route53 access_key_id and access_key_secret are required")
		}
		return NewRoute53Service(accessKeyID, accessKeySecret, region)

	case model.DNSProviderGoogleDNS:
		serviceAccountJSON := pickCredential(credentials, "service_account_json")
		if serviceAccountJSON == "" {
			return nil, errors.New("google_cloud_dns service_account_json is required")
		}
		return NewGoogleCloudDNSService(serviceAccountJSON)

	case model.DNSProviderBaiduDNS:
		accessKey := pickCredential(credentials, "access_key", "ak", "access_key_id")
		secretKey := pickCredential(credentials, "secret_key", "sk", "access_key_secret")
		if accessKey == "" || secretKey == "" {
			return nil, errors.New("baidu_cloud_dns access_key and secret_key are required")
		}
		return NewBaiduCloudDNSService(accessKey, secretKey)

	case model.DNSProviderDNSCom:
		apiID := pickCredential(credentials, "api_id")
		apiKey := pickCredential(credentials, "api_key")
		if apiID == "" || apiKey == "" {
			return nil, errors.New("dns_com api_id and api_key are required")
		}
		return NewDNSComService(apiID, apiKey)

	case model.DNSProviderDNSLA:
		apiID := pickCredential(credentials, "api_id")
		apiSecret := pickCredential(credentials, "api_secret")
		if apiID == "" || apiSecret == "" {
			return nil, errors.New("dnsla api_id and api_secret are required")
		}
		return NewDNSLAService(apiID, apiSecret)

	case model.DNSProviderWestCN:
		username := pickCredential(credentials, "username")
		password := pickCredential(credentials, "password")
		if username == "" || password == "" {
			return nil, errors.New("westcn_dns username and password are required")
		}
		return NewWestCNDNSService(username, password)

	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}

func pickCredential(credentials map[string]string, keys ...string) string {
	if len(credentials) == 0 {
		return ""
	}

	normalized := make(map[string]string, len(credentials))
	for k, v := range credentials {
		normalized[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	for _, key := range keys {
		if v := normalized[strings.ToLower(strings.TrimSpace(key))]; v != "" {
			return v
		}
	}
	return ""
}
