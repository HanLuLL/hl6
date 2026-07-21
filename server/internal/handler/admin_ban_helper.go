package handler

import (
	"context"
	"errors"
	"log"
	"time"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type adminBanExecutionResult struct {
	TargetRole        string
	SubdomainsDeleted int
	DeletedDNSCount   int
}

func executeAdminBanUserWithCleanup(
	ctx context.Context,
	repo *repository.Repository,
	ops *service.DNSOperationService,
	adminID uint,
	target *model.User,
	reason string,
	bannedUntil *time.Time,
) (adminBanExecutionResult, []cfFailureRecord, *uint, error) {
	result := adminBanExecutionResult{}
	if target == nil {
		return result, nil, nil, gorm.ErrRecordNotFound
	}

	result.TargetRole = target.Role

	subdomains, err := repo.ListSubdomainsByUserWithRecords(target.ID)
	if err != nil {
		return result, nil, nil, err
	}

	type deleteCandidate struct {
		sub model.Subdomain
		rec model.DNSRecord
	}

	candidates := make([]deleteCandidate, 0)
	failures := make([]cfFailureRecord, 0)
	accountErrCache := make(map[uint]error)

	for _, sub := range subdomains {
		accountID := sub.Domain.ProviderAccountID
		if _, ok := accountErrCache[accountID]; !ok {
			if accountID == 0 {
				accountErrCache[accountID] = errors.New("provider account id is empty")
			} else {
				_, accountErrCache[accountID] = repo.FindDNSProviderAccount(accountID)
			}
		}

		for _, rec := range sub.DNSRecords {
			if accountErrCache[accountID] != nil {
				failures = append(failures, cfFailureRecord{
					SubdomainFQDN:    sub.FQDN,
					RecordType:       rec.Type,
					RecordContent:    rec.Content,
					ProviderRecordID: rec.ProviderRecordID,
					Error:            accountErrCache[accountID].Error(),
				})
				continue
			}
			candidates = append(candidates, deleteCandidate{sub: sub, rec: rec})
		}
	}

	if len(failures) > 0 {
		return result, failures, nil, nil
	}

	batchItems := make([]service.BatchDeleteItem, 0, len(candidates))
	for _, item := range candidates {
		batchItems = append(batchItems, service.BatchDeleteItem{
			RecordID:          item.rec.ID,
			SubdomainFQDN:     item.sub.FQDN,
			Provider:          item.sub.Domain.Provider,
			ProviderAccountID: item.sub.Domain.ProviderAccountID,
			ZoneID:            item.sub.Domain.ProviderZoneID,
			ProviderRecordID:  item.rec.ProviderRecordID,
			RecordType:        item.rec.Type,
			Name:              item.rec.Name,
			Content:           item.rec.Content,
			TTL:               item.rec.TTL,
			Proxied:           item.rec.Proxied,
		})
	}

	if ctx == nil {
		ctx = context.Background()
	}
	deleteResult := ops.DeleteRecordsBatch(ctx, batchItems, 3)
	if deleteResult.Async {
		jobID := deleteResult.JobID
		return result, nil, &jobID, nil
	}
	if deleteResult.Failed > 0 {
		for _, f := range deleteResult.Failures {
			failures = append(failures, cfFailureRecord{
				SubdomainFQDN:    f.SubdomainFQDN,
				RecordType:       f.RecordType,
				RecordContent:    f.RecordContent,
				ProviderRecordID: f.ProviderRecordID,
				Error:            f.Error,
			})
		}
		result.DeletedDNSCount = deleteResult.Succeeded
		return result, failures, nil, nil
	}

	subdomainIDs := make([]uint, 0, len(subdomains))
	for _, sub := range subdomains {
		subdomainIDs = append(subdomainIDs, sub.ID)
	}

	now := time.Now()
	targetRole := target.Role

	if err := repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", adminBanGuardLockKey).Error; err != nil {
			return err
		}

		var lockedTarget model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedTarget, target.ID).Error; err != nil {
			return err
		}
		targetRole = lockedTarget.Role

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
			"banned_until":  bannedUntil,
			"banned_by":     adminID,
		}).Error; err != nil {
			return err
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
		return result, nil, nil, err
	}

	result.TargetRole = targetRole
	result.SubdomainsDeleted = len(subdomainIDs)
	result.DeletedDNSCount = deleteResult.Succeeded
	return result, nil, nil, nil
}

// executeAdminDeleteUserWithCleanup 完全删除用户及其所有关联数据
func executeAdminDeleteUserWithCleanup(
	ctx context.Context,
	repo *repository.Repository,
	ops *service.DNSOperationService,
	adminID uint,
	target *model.User,
) (adminBanExecutionResult, []cfFailureRecord, *uint, error) {
	result := adminBanExecutionResult{}
	if target == nil {
		return result, nil, nil, gorm.ErrRecordNotFound
	}

	result.TargetRole = target.Role

	// 1. 获取用户的所有子域名和 DNS 记录
	subdomains, err := repo.ListSubdomainsByUserWithRecords(target.ID)
	if err != nil {
		log.Printf("[admin] delete user %d: list subdomains failed: %v", target.ID, err)
		return result, nil, nil, err
	}

	type deleteCandidate struct {
		sub model.Subdomain
		rec model.DNSRecord
	}

	candidates := make([]deleteCandidate, 0)
	failures := make([]cfFailureRecord, 0)
	accountErrCache := make(map[uint]error)

	for _, sub := range subdomains {
		accountID := sub.Domain.ProviderAccountID
		if _, ok := accountErrCache[accountID]; !ok {
			if accountID == 0 {
				accountErrCache[accountID] = errors.New("provider account id is empty")
			} else {
				_, accountErrCache[accountID] = repo.FindDNSProviderAccount(accountID)
			}
		}

		for _, rec := range sub.DNSRecords {
			if accountErrCache[accountID] != nil {
				failures = append(failures, cfFailureRecord{
					SubdomainFQDN:    sub.FQDN,
					RecordType:       rec.Type,
					RecordContent:    rec.Content,
					ProviderRecordID: rec.ProviderRecordID,
					Error:            accountErrCache[accountID].Error(),
				})
				continue
			}
			candidates = append(candidates, deleteCandidate{sub: sub, rec: rec})
		}
	}

	if len(failures) > 0 {
		return result, failures, nil, nil
	}

	// 2. 批量删除 DNS 记录
	batchItems := make([]service.BatchDeleteItem, 0, len(candidates))
	for _, item := range candidates {
		batchItems = append(batchItems, service.BatchDeleteItem{
			RecordID:          item.rec.ID,
			SubdomainFQDN:     item.sub.FQDN,
			Provider:          item.sub.Domain.Provider,
			ProviderAccountID: item.sub.Domain.ProviderAccountID,
			ZoneID:            item.sub.Domain.ProviderZoneID,
			ProviderRecordID:  item.rec.ProviderRecordID,
			RecordType:        item.rec.Type,
			Name:              item.rec.Name,
			Content:           item.rec.Content,
			TTL:               item.rec.TTL,
			Proxied:           item.rec.Proxied,
		})
	}

	if ctx == nil {
		ctx = context.Background()
	}
	deleteResult := ops.DeleteRecordsBatch(ctx, batchItems, 3)
	if deleteResult.Async {
		jobID := deleteResult.JobID
		return result, nil, &jobID, nil
	}
	if deleteResult.Failed > 0 {
		for _, f := range deleteResult.Failures {
			failures = append(failures, cfFailureRecord{
				SubdomainFQDN:    f.SubdomainFQDN,
				RecordType:       f.RecordType,
				RecordContent:    f.RecordContent,
				ProviderRecordID: f.ProviderRecordID,
				Error:            f.Error,
			})
		}
		result.DeletedDNSCount = deleteResult.Succeeded
		return result, failures, nil, nil
	}

	// 3. 在事务中删除用户的所有数据
	subdomainIDs := make([]uint, 0, len(subdomains))
	for _, sub := range subdomains {
		subdomainIDs = append(subdomainIDs, sub.ID)
	}

	if err := repo.Transaction(func(tx *gorm.DB) error {
		// 删除 DNS 记录
		if len(subdomainIDs) > 0 {
			if err := tx.Where("subdomain_id IN ?", subdomainIDs).Delete(&model.DNSRecord{}).Error; err != nil {
				log.Printf("[admin] delete user %d: delete dns records failed: %v", target.ID, err)
				return err
			}
			// 删除子域名
			if err := tx.Where("id IN ?", subdomainIDs).Delete(&model.Subdomain{}).Error; err != nil {
				log.Printf("[admin] delete user %d: delete subdomains failed: %v", target.ID, err)
				return err
			}
		}

		// 删除积分余额和交易记录
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.CreditBalance{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete credit balance failed: %v", target.ID, err)
			return err
		}
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.CreditTransaction{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete credit transactions failed: %v", target.ID, err)
			return err
		}

		// 删除每日签到记录
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.DailyCheckinClaim{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete checkin claims failed: %v", target.ID, err)
			return err
		}

		// 删除用户会话
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.UserSession{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete sessions failed: %v", target.ID, err)
			return err
		}

		// 删除用户凭证
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.UserCredential{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete credential failed: %v", target.ID, err)
			return err
		}

		// 删除用户推荐关系（作为邀请者和被邀请者）
		if err := tx.Where("inviter_id = ? OR invitee_id = ?", target.ID, target.ID).Delete(&model.UserReferral{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete referrals failed: %v", target.ID, err)
			return err
		}

		// 删除申诉记录
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.UserAppeal{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete appeals failed: %v", target.ID, err)
			return err
		}

		// 删除通知已读回执
		// 注意：Notification 表是按 target_ids JSONB 数组分发的广播记录，没有 user_id 列，
		// 不能按 user_id 删除。被删用户 ID 残留在 target_ids 中无害（用户已不存在，查询不会匹配）。
		// 这里只删 NotificationRead（用户已读回执），符合预期。
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.NotificationRead{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete notification reads failed: %v", target.ID, err)
			return err
		}

		// 删除密码重置/账户激活令牌（不应残留，有安全风险）
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.AuthToken{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete auth tokens failed: %v", target.ID, err)
			return err
		}

		// 删除邮件发送记录（UserID 是可空指针，无外键约束，但清理避免残留）
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.EmailLog{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete email logs failed: %v", target.ID, err)
			return err
		}

		// 删除支付订单
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.PaymentOrder{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete payment orders failed: %v", target.ID, err)
			return err
		}

		// 删除审计日志（数据库外键约束 fk_audit_logs_user 要求必须删除，否则无法删 User）
		// 注意：管理员"删除用户"这个动作本身会另写一条 admin_delete_user 审计日志（ResourceID 保留被删用户 ID）
		if err := tx.Where("user_id = ?", target.ID).Delete(&model.AuditLog{}).Error; err != nil {
			log.Printf("[admin] delete user %d: delete audit logs failed: %v", target.ID, err)
			return err
		}

		// 最后删除用户本身
		if err := tx.Delete(&model.User{}, target.ID).Error; err != nil {
			log.Printf("[admin] delete user %d: delete user row failed: %v", target.ID, err)
			return err
		}

		return nil
	}); err != nil {
		log.Printf("[admin] delete user %d: transaction rolled back: %v", target.ID, err)
		return result, nil, nil, err
	}

	result.SubdomainsDeleted = len(subdomainIDs)
	result.DeletedDNSCount = deleteResult.Succeeded
	log.Printf("[admin] delete user %d: success, subdomains=%d dns=%d", target.ID, result.SubdomainsDeleted, result.DeletedDNSCount)
	return result, nil, nil, nil
}
