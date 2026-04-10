package handler

import (
	"errors"
	"fmt"
	"time"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type adminBanExecutionResult struct {
	TargetRole        string
	SubdomainsDeleted int
	QueueCount        int
}

func executeAdminBanUserWithCleanup(
	repo *repository.Repository,
	adminID uint,
	target *model.User,
	reason string,
) (adminBanExecutionResult, []cfFailureRecord, error) {
	result := adminBanExecutionResult{}
	if target == nil {
		return result, nil, gorm.ErrRecordNotFound
	}

	result.TargetRole = target.Role

	subdomains, err := repo.ListSubdomainsByUserWithRecords(target.ID)
	if err != nil {
		return result, nil, err
	}

	type deleteCandidate struct {
		sub model.Subdomain
		rec model.DNSRecord
	}

	candidates := make([]deleteCandidate, 0)
	failures := make([]cfFailureRecord, 0)
	accountErrCache := make(map[uint]error)

	for _, sub := range subdomains {
		accountID := sub.Domain.CloudflareAccountID
		if _, ok := accountErrCache[accountID]; !ok {
			if accountID == 0 {
				accountErrCache[accountID] = errors.New("cloudflare account id is empty")
			} else {
				_, accountErrCache[accountID] = repo.FindCloudflareAccount(accountID)
			}
		}

		for _, rec := range sub.DNSRecords {
			if accountErrCache[accountID] != nil {
				failures = append(failures, cfFailureRecord{
					SubdomainFQDN:      sub.FQDN,
					RecordType:         rec.Type,
					RecordContent:      rec.Content,
					CloudflareRecordID: rec.CloudflareRecordID,
					Error:              accountErrCache[accountID].Error(),
				})
				continue
			}
			candidates = append(candidates, deleteCandidate{sub: sub, rec: rec})
		}
	}

	if len(failures) > 0 {
		return result, failures, nil
	}

	subdomainIDs := make([]uint, 0, len(subdomains))
	for _, sub := range subdomains {
		subdomainIDs = append(subdomainIDs, sub.ID)
	}

	now := time.Now()
	targetRole := target.Role
	queuedOps := 0

	if err := repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", adminBanGuardLockKey).Error; err != nil {
			return err
		}

		var lockedTarget model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedTarget, target.ID).Error; err != nil {
			return err
		}
		targetRole = lockedTarget.Role

		// Safety guard: check and update in the same transaction to prevent concurrent bypass.
		if lockedTarget.Role == "admin" && !lockedTarget.IsBanned {
			var activeAdmins int64
			if err := tx.Model(&model.User{}).
				Where("role = ? AND is_banned = ?", "admin", false).
				Count(&activeAdmins).Error; err != nil {
				return err
			}
			if activeAdmins <= 1 {
				return errCannotBanLastActiveAdmin
			}
		}

		if err := tx.Model(&model.User{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
			"is_banned":     true,
			"banned_reason": reason,
			"banned_at":     now,
			"banned_by":     adminID,
		}).Error; err != nil {
			return err
		}

		for _, item := range candidates {
			if _, err := enqueueCloudflareTask(
				repo,
				tx,
				cloudflareTaskResourceDNSRecord,
				item.rec.ID,
				model.CloudflareTaskActionDeleteDNSRecord,
				model.CloudflareTaskPayload{
					CloudflareAccountID: item.sub.Domain.CloudflareAccountID,
					ZoneID:              item.sub.Domain.CloudflareZoneID,
					RecordID:            item.rec.CloudflareRecordID,
					RecordType:          item.rec.Type,
					Name:                item.rec.Name,
					Content:             item.rec.Content,
				},
				fmt.Sprintf("admin:ban:%d:%s", item.rec.ID, item.rec.CloudflareRecordID),
			); err != nil {
				return err
			}
			queuedOps++
		}

		if len(subdomainIDs) > 0 {
			if err := tx.Where("subdomain_id IN ?", subdomainIDs).Delete(&model.DNSRecord{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", subdomainIDs).Delete(&model.Subdomain{}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return result, nil, err
	}

	result.TargetRole = targetRole
	result.SubdomainsDeleted = len(subdomainIDs)
	result.QueueCount = queuedOps
	return result, nil, nil
}
