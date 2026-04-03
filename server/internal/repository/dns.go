package repository

import "hl6-server/internal/model"

func (r *Repository) ListDNSRecords(subdomainID uint) ([]model.DNSRecord, error) {
	var records []model.DNSRecord
	err := r.DB.Where("subdomain_id = ?", subdomainID).Order("type ASC, name ASC").Find(&records).Error
	return records, err
}

func (r *Repository) FindDNSRecord(id uint) (*model.DNSRecord, error) {
	var record model.DNSRecord
	err := r.DB.First(&record, id).Error
	return &record, err
}

func (r *Repository) CreateDNSRecord(record *model.DNSRecord) error {
	return r.DB.Create(record).Error
}

func (r *Repository) UpdateDNSRecord(record *model.DNSRecord) error {
	return r.DB.Save(record).Error
}

func (r *Repository) DeleteDNSRecord(id uint) error {
	return r.DB.Delete(&model.DNSRecord{}, id).Error
}

func (r *Repository) CountDNSRecordsBySubdomain(subdomainID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ?", subdomainID).Count(&count).Error
	return count, err
}

func (r *Repository) HasNonCNAMERecords(subdomainID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type != ?", subdomainID, "CNAME").Count(&count).Error
	return count > 0, err
}

func (r *Repository) HasCNAMERecord(subdomainID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type = ?", subdomainID, "CNAME").Count(&count).Error
	return count > 0, err
}

func (r *Repository) HasDuplicateRecord(subdomainID uint, recordType, content string) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type = ? AND content = ?", subdomainID, recordType, content).Count(&count).Error
	return count > 0, err
}

func (r *Repository) HasDuplicateRecordExcluding(subdomainID uint, recordType, content string, excludeID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type = ? AND content = ? AND id != ?", subdomainID, recordType, content, excludeID).Count(&count).Error
	return count > 0, err
}

type AdminDNSRecordDTO struct {
	model.DNSRecord
	UserID     uint   `json:"user_id"`
	UserEmail  string `json:"user_email"`
	UserName   string `json:"user_name"`
	DomainID   uint   `json:"domain_id"`
	DomainName string `json:"domain_name"`
}

func (r *Repository) AdminListDNSRecords(page, perPage int, search string, domainID, groupID *uint) ([]AdminDNSRecordDTO, int64, error) {
	var results []AdminDNSRecordDTO
	var total int64

	q := r.DB.Table("dns_records").
		Select("dns_records.*, subdomains.user_id, users.email as user_email, users.name as user_name, subdomains.domain_id, domains.name as domain_name").
		Joins("JOIN subdomains ON subdomains.id = dns_records.subdomain_id").
		Joins("JOIN users ON users.id = subdomains.user_id").
		Joins("JOIN domains ON domains.id = subdomains.domain_id")

	if search != "" {
		like := "%" + escapeLike(search) + "%"
		q = q.Where("dns_records.name ILIKE ? OR dns_records.content ILIKE ?", like, like)
	}
	if domainID != nil {
		q = q.Where("subdomains.domain_id = ?", *domainID)
	}
	if groupID != nil {
		q = q.Where("users.group_id = ?", *groupID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset((page - 1) * perPage).Limit(perPage).Order("dns_records.created_at DESC").Scan(&results).Error
	if results == nil {
		results = []AdminDNSRecordDTO{}
	}
	return results, total, err
}

func (r *Repository) FindDNSRecordWithSubdomain(id uint) (*model.DNSRecord, *model.Subdomain, error) {
	var record model.DNSRecord
	if err := r.DB.First(&record, id).Error; err != nil {
		return nil, nil, err
	}
	var sub model.Subdomain
	if err := r.DB.Preload("Domain").First(&sub, record.SubdomainID).Error; err != nil {
		return &record, nil, err
	}
	return &record, &sub, nil
}
