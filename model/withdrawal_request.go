package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	WithdrawalStatusPending   = "pending"
	WithdrawalStatusApproved  = "approved"
	WithdrawalStatusRejected  = "rejected"
	WithdrawalStatusCancelled = "cancelled"
)

type WithdrawalRequest struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	UserId       int    `json:"user_id" gorm:"type:int;index:idx_user_status,column:user_id"`
	Amount       int    `json:"amount" gorm:"type:int;column:amount"`
	Status       string `json:"status" gorm:"type:varchar(20);default:'pending';index:idx_user_status,column:status"`
	AdminId      int    `json:"admin_id" gorm:"type:int;default:0;column:admin_id"`
	RejectReason string `json:"reject_reason" gorm:"type:varchar(255);default:'';column:reject_reason"`
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (WithdrawalRequest) TableName() string {
	return "withdrawal_requests"
}

func GetWithdrawalRequestForUpdate(tx *gorm.DB, id int) (*WithdrawalRequest, error) {
	var req WithdrawalRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", id).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func GetWithdrawalRequestsByUserId(userId int, status string, page int, pageSize int) ([]WithdrawalRequest, int64, error) {
	var requests []WithdrawalRequest
	var total int64
	query := DB.Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Model(&WithdrawalRequest{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error; err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func GetWithdrawalRequestsAll(userId int, status string, page int, pageSize int) ([]WithdrawalRequest, int64, error) {
	var requests []WithdrawalRequest
	var total int64
	query := DB.Model(&WithdrawalRequest{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&requests).Error; err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func SumPendingAmountByUserId(userId int) (int, error) {
	statusCol := "`status`"
	if common.UsingPostgreSQL {
		statusCol = `"status"`
	}
	var result *int
	err := DB.Model(&WithdrawalRequest{}).
		Where("user_id = ? AND "+statusCol+" = ?", userId, WithdrawalStatusPending).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return *result, nil
}

func UpdateWithdrawalRequestStatus(tx *gorm.DB, id int, status string, adminId int, rejectReason string) error {
	updates := map[string]interface{}{
		"status":    status,
		"admin_id":  adminId,
	}
	if status == WithdrawalStatusRejected {
		updates["reject_reason"] = rejectReason
	}
	return tx.Model(&WithdrawalRequest{}).Where("id = ?", id).Updates(updates).Error
}
