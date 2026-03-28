package repository

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"hl6-server/internal/model"

	"gorm.io/gorm"
)

// notificationVisibilitySQL is the shared visibility predicate for notifications (table/alias n).
const notificationVisibilitySQL = `(
		(n.target_type = 'users' AND n.target_ids @> to_jsonb(?::bigint))
		OR (n.target_type = 'groups' AND n.target_ids @> to_jsonb(?::bigint) AND (n.visible_to_new = true OR n.created_at >= ?))
		OR (n.target_type = 'all' AND (n.visible_to_new = true OR n.created_at >= ?))
	)`

var notificationImageURLRegexp = regexp.MustCompile(`/api/v1/notifications/images/(\d+)`)

func parseNotificationImageIDsFromContent(content string) []uint {
	matches := notificationImageURLRegexp.FindAllStringSubmatch(content, -1)
	var imageIDs []uint
	for _, m := range matches {
		id, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			continue
		}
		imageIDs = append(imageIDs, uint(id))
	}
	return imageIDs
}

func linkPendingNotificationImages(tx *gorm.DB, notificationID uint, imageIDs []uint) error {
	if len(imageIDs) == 0 {
		return nil
	}
	return tx.Model(&model.NotificationImage{}).
		Where("id IN ? AND notification_id IS NULL", imageIDs).
		Update("notification_id", notificationID).Error
}

type NotificationWithRead struct {
	model.Notification
	IsRead bool `json:"is_read"`
}

func (r *Repository) ListNotificationsForUser(userID, groupID uint, userCreatedAt string, offset, limit int) ([]NotificationWithRead, int64, error) {
	var results []NotificationWithRead
	var total int64

	countSQL := `SELECT COUNT(*) FROM notifications n WHERE ` + notificationVisibilitySQL
	if err := r.DB.Raw(countSQL, userID, groupID, userCreatedAt, userCreatedAt).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []NotificationWithRead{}, 0, nil
	}

	querySQL := `SELECT n.id, n.title, n.content, n.type, n.target_type, n.target_ids, n.visible_to_new, n.created_by, n.created_at,
		CASE WHEN nr.id IS NOT NULL THEN true ELSE false END as is_read
		FROM notifications n
		LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?
		WHERE ` + notificationVisibilitySQL + `
		ORDER BY
			CASE WHEN nr.id IS NULL THEN 0 ELSE 1 END ASC,
			CASE
				WHEN nr.id IS NULL THEN
					CASE n.type WHEN 'urgent' THEN 0 WHEN 'pinned' THEN 1 ELSE 2 END
				ELSE
					CASE n.type WHEN 'pinned' THEN 0 ELSE 1 END
			END ASC,
			n.created_at DESC
		OFFSET ? LIMIT ?`

	err := r.DB.Raw(querySQL, userID, userID, groupID, userCreatedAt, userCreatedAt, offset, limit).Scan(&results).Error
	if results == nil {
		results = []NotificationWithRead{}
	}
	return results, total, err
}

func (r *Repository) FindNotificationForUser(id, userID, groupID uint, userCreatedAt string) (*NotificationWithRead, error) {
	var result NotificationWithRead

	querySQL := `SELECT n.*,
		CASE WHEN nr.id IS NOT NULL THEN true ELSE false END as is_read
		FROM notifications n
		LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?
		WHERE n.id = ? AND ` + notificationVisibilitySQL

	err := r.DB.Raw(querySQL, userID, id, userID, groupID, userCreatedAt, userCreatedAt).Scan(&result).Error
	if result.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &result, err
}

func (r *Repository) HasUnreadNotifications(userID, groupID uint, userCreatedAt string) (bool, error) {
	var count int64

	querySQL := `SELECT COUNT(*) FROM notifications n
		WHERE NOT EXISTS (SELECT 1 FROM notification_reads nr WHERE nr.notification_id = n.id AND nr.user_id = ?)
		AND ` + notificationVisibilitySQL

	err := r.DB.Raw(querySQL, userID, userID, groupID, userCreatedAt, userCreatedAt).Scan(&count).Error
	return count > 0, err
}

func (r *Repository) MarkNotificationRead(notificationID, userID uint) error {
	read := model.NotificationRead{
		NotificationID: notificationID,
		UserID:         userID,
	}
	return r.DB.Where("notification_id = ? AND user_id = ?", notificationID, userID).
		FirstOrCreate(&read).Error
}

func (r *Repository) CreateNotification(n *model.Notification) error {
	return r.DB.Create(n).Error
}

func (r *Repository) CreateNotificationWithImages(n *model.Notification) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(n).Error; err != nil {
			return err
		}
		imageIDs := parseNotificationImageIDsFromContent(n.Content)
		if err := linkPendingNotificationImages(tx, n.ID, imageIDs); err != nil {
			return fmt.Errorf("failed to link images: %w", err)
		}
		return nil
	})
}

func (r *Repository) DeleteNotification(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notification_id = ?", id).Delete(&model.NotificationRead{}).Error; err != nil {
			return err
		}
		if err := tx.Where("notification_id = ?", id).Delete(&model.NotificationImage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Notification{}, id).Error
	})
}

func (r *Repository) FindNotification(id uint) (*model.Notification, error) {
	var n model.Notification
	err := r.DB.Preload("Creator").First(&n, id).Error
	return &n, err
}

type NotificationWithReadCount struct {
	model.Notification
	ReadCount int64 `json:"read_count"`
}

func (r *Repository) ListNotificationsAdmin(page, perPage int) ([]NotificationWithReadCount, int64, error) {
	var total int64
	if err := r.DB.Model(&model.Notification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []NotificationWithReadCount
	err := r.DB.Model(&model.Notification{}).
		Select("notifications.*, (SELECT COUNT(*) FROM notification_reads nr WHERE nr.notification_id = notifications.id) as read_count").
		Preload("Creator").
		Offset((page - 1) * perPage).Limit(perPage).
		Order("created_at DESC").
		Find(&results).Error
	return results, total, err
}

func (r *Repository) UpdateNotificationWithImages(n *model.Notification) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(n).Select("title", "content", "type", "updated_at").Updates(n).Error; err != nil {
			return err
		}

		imageIDs := parseNotificationImageIDsFromContent(n.Content)

		if err := linkPendingNotificationImages(tx, n.ID, imageIDs); err != nil {
			return fmt.Errorf("failed to link images: %w", err)
		}

		delQuery := tx.Where("notification_id = ?", n.ID)
		if len(imageIDs) > 0 {
			delQuery = delQuery.Where("id NOT IN ?", imageIDs)
		}
		if err := delQuery.Delete(&model.NotificationImage{}).Error; err != nil {
			return fmt.Errorf("failed to delete orphaned images: %w", err)
		}

		return nil
	})
}

func (r *Repository) CreateNotificationImage(img *model.NotificationImage) error {
	return r.DB.Create(img).Error
}

func (r *Repository) FindNotificationImage(id uint) (*model.NotificationImage, error) {
	var img model.NotificationImage
	err := r.DB.First(&img, id).Error
	return &img, err
}

func (r *Repository) GetNotificationTargetUserIDs(n *model.Notification) ([]uint, error) {
	switch n.TargetType {
	case "users":
		var ids []uint
		if err := json.Unmarshal(n.TargetIDs, &ids); err != nil {
			return nil, err
		}
		return ids, nil
	case "groups":
		var groupIDs []uint
		if err := json.Unmarshal(n.TargetIDs, &groupIDs); err != nil {
			return nil, err
		}
		var userIDs []uint
		err := r.DB.Model(&model.User{}).Where("group_id IN ?", groupIDs).Pluck("id", &userIDs).Error
		return userIDs, err
	case "all":
		return nil, nil
	default:
		return nil, nil
	}
}
