package repository

import "hl6-server/internal/model"

func (r *Repository) CreatePaymentOrder(order *model.PaymentOrder) error {
	return r.DB.Create(order).Error
}

func (r *Repository) GetPaymentOrderByNo(orderNo string) (*model.PaymentOrder, error) {
	var order model.PaymentOrder
	err := r.DB.Where("order_no = ?", orderNo).First(&order).Error
	return &order, err
}

func (r *Repository) UpdatePaymentOrder(order *model.PaymentOrder) error {
	return r.DB.Save(order).Error
}
