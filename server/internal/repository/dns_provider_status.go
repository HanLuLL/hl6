package repository

import (
	"time"

	"hl6-server/internal/model"
)

// DNSProviderStatusEntry aggregates health information for a single provider.
type DNSProviderStatusEntry struct {
	Provider            string     `json:"provider"`
	AccountsTotal       int64      `json:"accounts_total"`
	AccountsActive      int64      `json:"accounts_active"`
	LastVerifiedAt      *time.Time `json:"last_verified_at"`
	LastFailureAt       *time.Time `json:"last_failure_at"`
	LastFailureCategory string     `json:"last_failure_category"`
	FailureCount24h     int64      `json:"failure_count_24h"`
	MigrationQueueSize  int64      `json:"migration_queue_size"`
	Health              string     `json:"health"` // healthy / degraded / unhealthy
}

// GetDNSProviderStatus returns aggregated health status for all providers.
func (r *Repository) GetDNSProviderStatus() ([]DNSProviderStatusEntry, error) {
	// Get all accounts grouped by provider
	type accountRow struct {
		Provider       string
		Total          int64
		Active         int64
		LastVerifiedAt *time.Time
	}
	var accountRows []accountRow
	if err := r.DB.Model(&model.DNSProviderAccount{}).
		Select("provider, COUNT(*) as total, SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active, MAX(last_verified_at) as last_verified_at").
		Group("provider").
		Scan(&accountRows).Error; err != nil {
		return nil, err
	}

	// Get failure events in last 24h per provider
	type failureRow struct {
		Provider            string
		FailureCount        int64
		LastFailureAt       *time.Time
		LastFailureCategory string
	}
	since := time.Now().Add(-24 * time.Hour)
	var failureCountRows []failureRow
	if err := r.DB.Model(&model.DNSOperationEvent{}).
		Select("provider, COUNT(*) as failure_count").
		Where("success = false AND created_at >= ? AND provider != ''", since).
		Group("provider").
		Scan(&failureCountRows).Error; err != nil {
		return nil, err
	}
	failureMap := make(map[string]failureRow, len(failureCountRows))
	for _, f := range failureCountRows {
		failureMap[f.Provider] = f
	}

	var latestFailureRows []failureRow
	if err := r.DB.Model(&model.DNSOperationEvent{}).
		Select("DISTINCT ON (provider) provider, created_at as last_failure_at, error_category as last_failure_category").
		Where("success = false AND created_at >= ? AND provider != ''", since).
		Order("provider, created_at DESC, id DESC").
		Scan(&latestFailureRows).Error; err != nil {
		return nil, err
	}
	for _, latest := range latestFailureRows {
		row := failureMap[latest.Provider]
		row.Provider = latest.Provider
		row.LastFailureAt = latest.LastFailureAt
		row.LastFailureCategory = latest.LastFailureCategory
		failureMap[latest.Provider] = row
	}

	// Get migration queue sizes per provider (via target_provider)
	type queueRow struct {
		TargetProvider string
		QueueSize      int64
	}
	var queueRows []queueRow
	if err := r.DB.Model(&model.DomainDNSMigrationTask{}).
		Select("target_provider, COUNT(*) as queue_size").
		Where("status IN ?", []string{model.MigrationTaskStatusPending, model.MigrationTaskStatusRunning}).
		Group("target_provider").
		Scan(&queueRows).Error; err != nil {
		return nil, err
	}
	queueMap := make(map[string]int64, len(queueRows))
	for _, q := range queueRows {
		queueMap[q.TargetProvider] = q.QueueSize
	}

	result := make([]DNSProviderStatusEntry, 0, len(accountRows))
	for _, ar := range accountRows {
		fr := failureMap[ar.Provider]
		entry := DNSProviderStatusEntry{
			Provider:            ar.Provider,
			AccountsTotal:       ar.Total,
			AccountsActive:      ar.Active,
			LastVerifiedAt:      ar.LastVerifiedAt,
			LastFailureAt:       fr.LastFailureAt,
			LastFailureCategory: fr.LastFailureCategory,
			FailureCount24h:     fr.FailureCount,
			MigrationQueueSize:  queueMap[ar.Provider],
		}
		// Determine health
		switch {
		case ar.Active == 0:
			entry.Health = "unhealthy"
		case fr.FailureCount > 10 || ar.Active < ar.Total:
			entry.Health = "degraded"
		default:
			entry.Health = "healthy"
		}
		result = append(result, entry)
	}
	return result, nil
}
