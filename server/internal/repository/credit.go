package repository

import (
	"encoding/json"
	"errors"

	"hl6-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) GetCreditBalance(userID uint) (*model.CreditBalance, error) {
	var balance model.CreditBalance
	err := r.DB.Where("user_id = ?", userID).First(&balance).Error
	return &balance, err
}

func (r *Repository) EnsureCreditBalance(userID uint) (*model.CreditBalance, error) {
	var balance model.CreditBalance
	err := r.DB.Where("user_id = ?", userID).FirstOrCreate(&balance, model.CreditBalance{UserID: userID, Balance: 0}).Error
	return &balance, err
}

func (r *Repository) DeductCredits(tx *gorm.DB, userID uint, amount model.Credit, descriptionKey string, descriptionParams json.RawMessage) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		return err
	}
	if balance.Balance < amount {
		return gorm.ErrInvalidData
	}
	balance.Balance -= amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:            userID,
		Amount:            -amount,
		Type:              "deduct",
		DescriptionKey:    descriptionKey,
		DescriptionParams: descriptionParams,
		BalanceAfter:      balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) GrantCredits(tx *gorm.DB, userID uint, amount model.Credit, descriptionKey string, descriptionParams json.RawMessage) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		balance = model.CreditBalance{UserID: userID, Balance: 0}
		if err := tx.Create(&balance).Error; err != nil {
			return err
		}
	}
	balance.Balance += amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:            userID,
		Amount:            amount,
		Type:              "grant",
		DescriptionKey:    descriptionKey,
		DescriptionParams: descriptionParams,
		BalanceAfter:      balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) RefundCredits(tx *gorm.DB, userID uint, amount model.Credit, descriptionKey string, descriptionParams json.RawMessage) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		return err
	}
	balance.Balance += amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:            userID,
		Amount:            amount,
		Type:              "refund",
		DescriptionKey:    descriptionKey,
		DescriptionParams: descriptionParams,
		BalanceAfter:      balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) ListTransactions(userID uint, page, perPage int) ([]model.CreditTransaction, int64, error) {
	var txns []model.CreditTransaction
	var total int64
	q := r.DB.Model(&model.CreditTransaction{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&txns).Error
	return txns, total, err
}
