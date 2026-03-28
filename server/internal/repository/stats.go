package repository

import "hl6-server/internal/model"

func (r *Repository) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)
	var users, domains, subdomains, dnsRecords, userGroups int64
	if err := r.DB.Model(&model.User{}).Count(&users).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Model(&model.Domain{}).Where("is_active = ?", true).Count(&domains).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Model(&model.Subdomain{}).Count(&subdomains).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Model(&model.DNSRecord{}).Count(&dnsRecords).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Model(&model.UserGroup{}).Count(&userGroups).Error; err != nil {
		return nil, err
	}
	stats["users"] = users
	stats["domains"] = domains
	stats["subdomains"] = subdomains
	stats["dns_records"] = dnsRecords
	stats["user_groups"] = userGroups
	return stats, nil
}
