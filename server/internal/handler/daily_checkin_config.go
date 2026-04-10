package handler

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"
)

const (
	configKeyDailyCheckinEnabled  = "daily_checkin_enabled"
	configKeyDailyCheckinCredits  = "daily_checkin_credits"
	configKeyDailyCheckinGroupIDs = "daily_checkin_group_ids"
)

var dailyCheckinCreditPattern = regexp.MustCompile(`^\d+(\.\d)?$`)
var errDailyCheckinGroupsNotFound = errors.New("daily checkin group ids not found")

var beijingLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("UTC+8", 8*60*60)
	}
	return loc
}()

type DailyCheckinRuntimeConfig struct {
	Enabled         bool
	Reward          model.Credit
	AllowedGroupIDs []uint
	allowedGroupSet map[uint]struct{}
}

func (c *DailyCheckinRuntimeConfig) IsGroupAllowed(groupID uint) bool {
	if c == nil || len(c.allowedGroupSet) == 0 {
		return false
	}
	_, ok := c.allowedGroupSet[groupID]
	return ok
}

func loadDailyCheckinRuntimeConfig(repo *repository.Repository) (*DailyCheckinRuntimeConfig, error) {
	configs, err := repo.GetSystemConfigsByKeys([]string{
		configKeyDailyCheckinEnabled,
		configKeyDailyCheckinCredits,
		configKeyDailyCheckinGroupIDs,
	})
	if err != nil {
		return nil, err
	}

	enabled := strings.EqualFold(strings.TrimSpace(configs[configKeyDailyCheckinEnabled]), "true")
	reward := parseDailyCheckinRewardForRuntime(configs[configKeyDailyCheckinCredits])
	groupIDs, err := parseDailyCheckinGroupIDs(configs[configKeyDailyCheckinGroupIDs])
	if err != nil {
		// Fail closed if config is malformed.
		groupIDs = []uint{}
	}

	allowed := make(map[uint]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		allowed[id] = struct{}{}
	}

	return &DailyCheckinRuntimeConfig{
		Enabled:         enabled && reward > 0,
		Reward:          reward,
		AllowedGroupIDs: groupIDs,
		allowedGroupSet: allowed,
	}, nil
}

func todayInBeijing(now time.Time) string {
	return now.In(beijingLocation).Format("2006-01-02")
}

func normalizeBooleanConfig(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "true", "false":
		return trimmed, nil
	default:
		return "", fmt.Errorf("invalid bool config value %q", raw)
	}
}

func normalizeDailyCheckinCredits(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "0"
	}
	if !dailyCheckinCreditPattern.MatchString(trimmed) {
		return "", fmt.Errorf("invalid daily checkin credits %q", raw)
	}

	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || f < 0 {
		return "", fmt.Errorf("invalid daily checkin credits %q", raw)
	}
	return formatCreditConfigValue(model.CreditFromFloat(f)), nil
}

func parseDailyCheckinGroupIDs(raw string) ([]uint, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []uint{}, nil
	}

	parts := strings.Split(trimmed, ",")
	seen := make(map[uint]struct{}, len(parts))
	ids := make([]uint, 0, len(parts))

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		parsed, err := strconv.ParseUint(token, 10, 64)
		if err != nil || parsed == 0 {
			return nil, fmt.Errorf("invalid group id %q", token)
		}

		id := uint(parsed)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func formatDailyCheckinGroupIDs(ids []uint) string {
	if len(ids) == 0 {
		return ""
	}
	normalized := make([]uint, len(ids))
	copy(normalized, ids)
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })

	parts := make([]string, 0, len(normalized))
	var prev uint
	for idx, id := range normalized {
		if idx > 0 && id == prev {
			continue
		}
		parts = append(parts, strconv.FormatUint(uint64(id), 10))
		prev = id
	}
	return strings.Join(parts, ",")
}

func validateDailyCheckinGroupsExist(repo *repository.Repository, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := repo.GetDB().Model(&model.UserGroup{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if int(count) != len(ids) {
		return errDailyCheckinGroupsNotFound
	}
	return nil
}

func parseDailyCheckinRewardForRuntime(raw string) model.Credit {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return model.CreditFromFloat(f)
}

func formatCreditConfigValue(amount model.Credit) string {
	if amount <= 0 {
		return "0"
	}
	if amount%10 == 0 {
		return strconv.FormatInt(int64(amount/10), 10)
	}
	return strconv.FormatFloat(amount.ToFloat(), 'f', 1, 64)
}
